//go:build windows

// Package secrets keeps Arion's local encryption key in the OS credential
// store: the login Keychain on macOS, DPAPI on Windows (PRV-04).
package secrets

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	procLocalFree          = kernel32.NewProc("LocalFree")
)

type blob struct {
	size uint32
	data *byte
}

func newBlob(b []byte) *blob { return &blob{size: uint32(len(b)), data: &b[0]} }

func (b *blob) bytes() []byte { return append([]byte(nil), unsafe.Slice(b.data, b.size)...) }

const keyBytes = 32

// Key returns the named key protected by DPAPI for the current user. The
// protected blob lives in %AppData%\Arion; it is useless to other users.
func Key(service, account string) ([]byte, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(base, service, account+".dpapi")
	if sealed, err := os.ReadFile(path); err == nil {
		var out blob
		if r, _, _ := procCryptUnprotectData.Call(uintptr(unsafe.Pointer(newBlob(sealed))), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&out))); r == 0 {
			return nil, errors.New("DPAPI unprotect failed")
		}
		defer procLocalFree.Call(uintptr(unsafe.Pointer(out.data)))
		key := out.bytes()
		if len(key) != keyBytes {
			return nil, errors.New("protected key has unexpected size")
		}
		return key, nil
	}
	key := make([]byte, keyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	var out blob
	if r, _, _ := procCryptProtectData.Call(uintptr(unsafe.Pointer(newBlob(key))), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&out))); r == 0 {
		return nil, errors.New("DPAPI protect failed")
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.data)))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, out.bytes(), 0o600); err != nil {
		return nil, err
	}
	return key, nil
}
