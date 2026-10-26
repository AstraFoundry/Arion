package gguf

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Windows ignores POSIX mode bits. Create the socket directory with a
// protected inheritable DACL granting access only to the current user.
func privateDirectory() (string, error) {
	var token syscall.Token
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		return "", err
	}
	if err := syscall.OpenProcessToken(process, syscall.TOKEN_QUERY, &token); err != nil {
		return "", err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	sid, err := user.User.Sid.String()
	if err != nil {
		return "", err
	}
	sddl, err := syscall.UTF16PtrFromString("D:P(A;OICI;FA;;;" + sid + ")")
	if err != nil {
		return "", err
	}
	var descriptor uintptr
	convert := syscall.NewLazyDLL("advapi32.dll").NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
	if ok, _, err := convert.Call(uintptr(unsafe.Pointer(sddl)), 1, uintptr(unsafe.Pointer(&descriptor)), 0); ok == 0 {
		return "", err
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	dir := filepath.Join(os.TempDir(), "arion-llm-"+hex.EncodeToString(nonce[:]))
	name, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return "", err
	}
	attributes := syscall.SecurityAttributes{Length: uint32(unsafe.Sizeof(syscall.SecurityAttributes{})), SecurityDescriptor: descriptor}
	if err := syscall.CreateDirectory(name, &attributes); err != nil {
		return "", err
	}
	// AF_UNIX has a 108-byte address limit even when Windows accepts a much
	// longer TEMP path. Use the filesystem's canonical short path when present.
	short := make([]uint16, 32768)
	n, err := syscall.GetShortPathName(name, &short[0], uint32(len(short)))
	if err != nil || n >= uint32(len(short)) {
		_ = os.RemoveAll(dir)
		if err != nil {
			return "", err
		}
		return "", syscall.ERROR_INSUFFICIENT_BUFFER
	}
	return syscall.UTF16ToString(short[:n]), nil
}
