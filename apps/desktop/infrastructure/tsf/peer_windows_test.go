//go:build windows

package tsf

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/imebridge"
)

// Historical artifact fixtures exercise file/module provenance only. Their
// native protocols are intentionally refused for production voice capture.
const legacyVoiceBuild = "26213df3fd6f65108d402f4658b6c5e4dcc02cd705322019331e5eccb350e891"
const legacyKeyboardBuild = "f9f9485bc5bb1e46a656a0f418be5309af652660e076ecf419e877ef2b7e5b73"

func TestExplicitProtocolCompatibilityPolicy(t *testing.T) {
	for _, tc := range []struct {
		build              string
		protocol           int
		keyboard, accepted bool
	}{
		{legacyVoiceBuild, 0, false, false}, {legacyKeyboardBuild, 0, false, false},
		{legacyVoiceBuild, 1, false, false}, {legacyKeyboardBuild, 1, false, false},
		{"55fc16baa7b05a2a9b65d96e91631df9c75bbd605c2b2fc9e1cd2f519347eb88", 0, false, false},
		{strings.Repeat("a", 64), 0, false, false}, {strings.Repeat("a", 64), 1, false, false},
		{strings.Repeat("a", 64), 2, true, true}, {strings.Repeat("b", 64), 2, true, true},
		{strings.Repeat("a", 64), 3, false, false},
	} {
		keyboard, err := clientProtocol(imebridge.ClientHello{Build: tc.build, Protocol: tc.protocol})
		if (err == nil) != tc.accepted || err == nil && keyboard != tc.keyboard {
			t.Fatalf("policy %v: keyboard=%v err=%v", tc, keyboard, err)
		}
		if err != nil && domain.CodeOf(err) != domain.CodeContractUnsupported {
			t.Fatalf("unknown protocol error category: %v", err)
		}
	}
}

func TestProtectedClientRejectsUserOwnedAndWrongVersionPaths(t *testing.T) {
	root := t.TempDir()
	build := strings.Repeat("a", 64)
	directory := filepath.Join(root, build)
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(directory, "ArionTSF.dll")
	if err := os.WriteFile(file, []byte("untrusted"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateClientModule(root, file, build, false); err == nil {
		t.Fatal("user-owned synthetic DLL accepted")
	}
	if err := validateClientModule(root, file, strings.Repeat("b", 64), false); err == nil {
		t.Fatal("wrong version directory accepted")
	}
	if err := protectedClientACL(file); err == nil {
		t.Fatal("ordinary user owns trusted payload")
	}
}

func TestClientModuleClaimRemainsExactWithRetainedLegacyMapping(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Arion", "InputMethod")
	build := strings.Repeat("a", 64)
	current := filepath.Join(root, build, "ArionTSF.dll")
	legacy := filepath.Join(root, legacyVoiceBuild, "ArionTSF.dll")
	for _, tc := range []struct {
		name    string
		modules []string
		claim   string
		ok      bool
	}{
		{"current with cached legacy", []string{legacy, current}, build, true},
		{"current before cached legacy", []string{current, legacy}, build, true},
		{"claim absent", []string{legacy}, build, false},
		{"different advertised hash", []string{current}, strings.Repeat("b", 64), false},
		{"different root", []string{filepath.Join(root+"-other", build, "ArionTSF.dll")}, build, false},
		{"ambiguous claim", []string{current, strings.ToUpper(current)}, build, false},
		{"invalid identity", []string{current}, "../" + build, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module, err := claimedClientModule(root, tc.modules, tc.claim)
			if (err == nil) != tc.ok || tc.ok && module != current {
				t.Fatalf("module=%q error=%v", module, err)
			}
		})
	}
}

func TestRetainedInstalledClientModuleProof(t *testing.T) {
	root, err := installedClientRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, legacyKeyboardBuild, "ArionTSF.dll")
	if _, err = os.Stat(path); os.IsNotExist(err) {
		if os.Getenv("ARION_REQUIRE_RETAINED_CLIENT") == "1" {
			t.Fatal("required retained f9 fixture is absent")
		}
		t.Skip("reviewed retained f9 fixture is not installed; no installation is performed")
	}
	started := time.Now()
	if err = validateClientModule(root, path, legacyKeyboardBuild, true); err != nil {
		t.Fatal(err)
	}
	t.Logf("protected f9 full closure validation: %s", time.Since(started))
	manifest, err := os.ReadFile(filepath.Join(filepath.Dir(path), "keyboard", "runtime.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained module=%s SHA256=%s keyboardManifestSHA256=%x", path, legacyKeyboardBuild, sha256.Sum256(manifest))
	// Loading only DllMain does not activate TSF, register a profile or open a
	// user's keyboard vault. This exercises the actual kernel module inventory.
	name, _ := syscall.UTF16PtrFromString(path)
	handle, _, failure := peerKernelDLL.NewProc("LoadLibraryExW").Call(uintptr(unsafe.Pointer(name)), 0, 0x100|0x800)
	if handle == 0 {
		t.Fatal(failure)
	}
	defer syscall.FreeLibrary(syscall.Handle(handle))
	modules, err := loadedClientModules(uint32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, module := range modules {
		if strings.EqualFold(module, path) {
			found = true
		}
	}
	if !found {
		t.Fatalf("loaded protected DLL missing from kernel enumeration: %v", modules)
	}
	pipePath := fmt.Sprintf(`\\.\pipe\Arion-peer-proof-%d-%d`, os.Getpid(), time.Now().UnixNano())
	bridge := imebridge.New(pipePath, nil, nil)
	verified := make(chan error, 1)
	bridge.RequireClientVerification(func(peer imebridge.PeerIdentity, hello imebridge.ClientHello) error {
		started := time.Now()
		err := verifyWindowsClient(peer, hello)
		t.Logf("actual pipe identity and historical protocol rejection: %s", time.Since(started))
		verified <- err
		return err
	})
	if err = bridge.Listen(); err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	pipeName, _ := syscall.UTF16PtrFromString(pipePath)
	client, err := syscall.CreateFile(pipeName, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	file := os.NewFile(uintptr(client), "isolated-proof-pipe")
	defer file.Close()
	if err = json.NewEncoder(file).Encode(map[string]any{"op": "hello", "build": legacyKeyboardBuild, "protocol": 0}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-verified:
		if domain.CodeOf(err) != domain.CodeContractUnsupported {
			t.Fatalf("historical f9 must not satisfy the protocol-2 voice safety contract: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("real peer verification timed out")
	}
}
