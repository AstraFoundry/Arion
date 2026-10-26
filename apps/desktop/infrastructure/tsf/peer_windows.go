//go:build windows

package tsf

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/imebridge"
)

// Protocol 2 requires refusing both IS_PASSWORD and IS_PRIVATE text stores.
// Older contracts do not provide that safety guarantee, regardless of build.
// Compatible protocol-2 updates still need the complete protected closure;
// their DLL hash need not match the current application's bundled DLL hash.
func clientProtocol(hello imebridge.ClientHello) (needsKeyboard bool, err error) {
	if hello.Protocol == 2 {
		return true, nil
	}
	return false, domain.Fail(domain.CodeContractUnsupported, domain.DetailInputMethodProtocolUnsupported)
}

var peerKernelDLL = syscall.NewLazyDLL("kernel32.dll")
var peerSnapshot = peerKernelDLL.NewProc("CreateToolhelp32Snapshot")
var peerModuleFirst = peerKernelDLL.NewProc("Module32FirstW")
var peerModuleNext = peerKernelDLL.NewProc("Module32NextW")
var peerSecurity = syscall.NewLazyDLL("advapi32.dll").NewProc("GetNamedSecurityInfoW")
var peerGetACE = syscall.NewLazyDLL("advapi32.dll").NewProc("GetAce")
var peerLocalFree = peerKernelDLL.NewProc("LocalFree")
var peerKnownFolder = syscall.NewLazyDLL("shell32.dll").NewProc("SHGetKnownFolderPath")
var peerTaskFree = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")

type moduleEntry struct {
	Size, ModuleID, ProcessID, GlobalCount, ProcessCount uint32
	Base                                                 uintptr
	BaseSize                                             uint32
	Module                                               syscall.Handle
	Name                                                 [256]uint16
	Path                                                 [260]uint16
}

func loadedClientModules(pid uint32) ([]string, error) {
	var snapshot uintptr
	var failure error
	// The documented ERROR_BAD_LENGTH case is retried for a changing loader
	// list. Other failures remain verification failures, not empty results.
	for attempt := 0; attempt < 4; attempt++ {
		snapshot, _, failure = peerSnapshot.Call(0x8|0x10, uintptr(pid))
		if snapshot != ^uintptr(0) || !errors.Is(failure, syscall.Errno(24)) {
			break
		}
	}
	if snapshot == ^uintptr(0) {
		return nil, failure
	}
	defer syscall.CloseHandle(syscall.Handle(snapshot))
	entry := moduleEntry{}
	entry.Size = uint32(unsafe.Sizeof(entry))
	ok, _, failure := peerModuleFirst.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ok == 0 {
		return nil, failure
	}
	var result []string
	for {
		if strings.EqualFold(syscall.UTF16ToString(entry.Name[:]), "ArionTSF.dll") {
			result = append(result, syscall.UTF16ToString(entry.Path[:]))
		}
		ok, _, failure = peerModuleNext.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ok == 0 {
			if !errors.Is(failure, syscall.ERROR_NO_MORE_FILES) {
				return nil, failure
			}
			return result, nil
		}
	}
}

func installedClientRoot() (string, error) {
	id := syscall.GUID{Data1: 0x905e63b6, Data2: 0xc1bf, Data3: 0x494e, Data4: [8]byte{0xb2, 0x9c, 0x65, 0xb7, 0x32, 0xd3, 0xd2, 0x1a}}
	var text *uint16
	status, _, _ := peerKnownFolder.Call(uintptr(unsafe.Pointer(&id)), 0, 0, uintptr(unsafe.Pointer(&text)))
	if int32(status) < 0 || text == nil {
		return "", errors.New("known program directory unavailable")
	}
	defer peerTaskFree.Call(uintptr(unsafe.Pointer(text)))
	value := unsafe.Slice(text, 32768)
	return filepath.Join(syscall.UTF16ToString(value), "Arion", "InputMethod"), nil
}

func protectedClientACL(path string) error {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	var owner *syscall.SID
	var dacl, descriptor unsafe.Pointer
	status, _, _ := peerSecurity.Call(uintptr(unsafe.Pointer(name)), 1, 0x1|0x4, uintptr(unsafe.Pointer(&owner)), 0, uintptr(unsafe.Pointer(&dacl)), 0, uintptr(unsafe.Pointer(&descriptor)))
	if descriptor != nil {
		defer peerLocalFree.Call(uintptr(descriptor))
	}
	if status != 0 || owner == nil || dacl == nil {
		return errors.New("protected payload security unavailable")
	}
	privileged := func(sid *syscall.SID) bool {
		value, e := sid.String()
		return e == nil && (value == "S-1-5-18" || value == "S-1-5-32-544")
	}
	if !privileged(owner) {
		return errors.New("payload owner is not privileged")
	}
	type acl struct {
		Revision, Reserved     byte
		Size, Count, Reserved2 uint16
	}
	count := (*acl)(dacl).Count
	for i := uint16(0); i < count; i++ {
		var ace unsafe.Pointer
		if ok, _, _ := peerGetACE.Call(uintptr(dacl), uintptr(i), uintptr(unsafe.Pointer(&ace))); ok == 0 || ace == nil {
			return errors.New("payload access rule unavailable")
		}
		type header struct {
			Kind, Flags byte
			Size        uint16
			Mask        uint32
		}
		rule := (*header)(ace)
		if rule.Flags&0x8 != 0 || rule.Kind == 1 {
			continue
		} // inherit-only or deny
		if rule.Kind != 0 || rule.Size < 16 {
			return errors.New("unsupported payload access rule")
		}
		const writes = 0x2 | 0x4 | 0x10 | 0x40 | 0x100 | 0x10000 | 0x40000 | 0x80000 | 0x40000000 | 0x10000000
		if rule.Mask&writes != 0 && !privileged((*syscall.SID)(unsafe.Add(ace, 8))) {
			return errors.New("payload allows unprivileged replacement")
		}
	}
	return nil
}

func protectedClientEntry(path string) error {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)
	var info syscall.ByHandleFileInformation
	if err = syscall.GetFileInformationByHandle(handle, &info); err != nil {
		return err
	}
	if info.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&syscall.FILE_ATTRIBUTE_DIRECTORY == 0 && info.NumberOfLinks != 1 {
		return errors.New("linked protected payload")
	}
	return protectedClientACL(path)
}

func validateClientModule(root, path, build string, needsKeyboard bool) error {
	if len(build) != 64 || strings.ToLower(build) != build {
		return errors.New("invalid module identity")
	}
	if _, err := hex.DecodeString(build); err != nil {
		return err
	}
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	expected := filepath.Join(root, build, "ArionTSF.dll")
	if !strings.EqualFold(path, expected) {
		return errors.New("module path is outside its protected version")
	}
	// Inspect ancestors for redirects, including the fixed Program Files path;
	// strict Administrator/SYSTEM ownership starts at Arion's own directory.
	for parent := path; ; parent = filepath.Dir(parent) {
		name, err := syscall.UTF16PtrFromString(parent)
		if err != nil {
			return err
		}
		flags, err := syscall.GetFileAttributes(name)
		if err != nil || flags&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return errors.New("redirected module ancestor")
		}
		if strings.EqualFold(parent, filepath.Dir(root)) || strings.EqualFold(parent, root) || strings.EqualFold(parent, filepath.Dir(path)) || strings.EqualFold(parent, path) {
			if err := protectedClientEntry(parent); err != nil {
				return err
			}
		}
		if filepath.Dir(parent) == parent {
			break
		}
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(bytes)
	if hex.EncodeToString(digest[:]) != build {
		return errors.New("loaded module identity differs")
	}
	if !needsKeyboard {
		return nil
	}
	keyboard := filepath.Join(filepath.Dir(path), "keyboard")
	manifest, err := os.ReadFile(filepath.Join(keyboard, "runtime.json"))
	if err != nil {
		return err
	}
	digest = sha256.Sum256(manifest)
	if err = validateKeyboardRuntime(keyboard, hex.EncodeToString(digest[:])); err != nil {
		return err
	}
	return filepath.WalkDir(keyboard, func(path string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		return protectedClientEntry(path)
	})
}

func claimedClientModule(root string, modules []string, build string) (string, error) {
	if len(build) != 64 || strings.ToLower(build) != build {
		return "", errors.New("invalid module identity")
	}
	if _, err := hex.DecodeString(build); err != nil {
		return "", err
	}
	expected := filepath.Join(root, build, "ArionTSF.dll")
	var claimed string
	for _, module := range modules {
		if strings.EqualFold(filepath.Clean(module), expected) {
			if claimed != "" {
				return "", errors.New("ambiguous loaded module identity")
			}
			claimed = module
		}
	}
	if claimed == "" {
		return "", errors.New("claimed core is not loaded in the pipe client")
	}
	return claimed, nil
}

func verifyWindowsClient(peer imebridge.PeerIdentity, hello imebridge.ClientHello) error {
	artifact := func() error {
		return domain.Fail(domain.CodeArtifactUnverified, domain.DetailInputMethodArtifactUnverified)
	}
	if !peer.Alive() {
		return artifact()
	}
	needsKeyboard, err := clientProtocol(hello)
	if err != nil {
		return err
	}
	root, err := installedClientRoot()
	if err != nil {
		return artifact()
	}
	modules, err := loadedClientModules(peer.ProcessID)
	if err != nil {
		return artifact()
	}
	// COM may retain a legacy mapping after the stable loader activates a new
	// core. Verify the uniquely claimed loaded core, never an arbitrary first
	// module or the currently registered build. Old protocols remain refused.
	module, err := claimedClientModule(root, modules, hello.Build)
	if err != nil {
		return artifact()
	}
	if err = validateClientModule(root, module, hello.Build, needsKeyboard); err != nil {
		return artifact()
	}
	if !peer.Alive() {
		return artifact()
	}
	return nil
}
