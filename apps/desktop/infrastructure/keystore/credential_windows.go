//go:build windows

package keystore

import (
	"runtime"
	"syscall"
	"unsafe"

	"arion.local/desktop/internal/account"
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
	errorNotFound           = 1168
)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredRead   = advapi32.NewProc("CredReadW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

// credential matches CREDENTIALW. The constants fail the build when padding
// drifts from the Windows layout.
type credential struct {
	flags          uint32
	typ            uint32
	targetName     *uint16
	comment        *uint16
	lastWritten    syscall.Filetime
	blobSize       uint32
	blob           *byte
	persist        uint32
	attributeCount uint32
	attributes     uintptr
	targetAlias    *uint16
	userName       *uint16
}

const (
	_ = uint64(40 - unsafe.Offsetof(credential{}.blob))
	_ = uint64(unsafe.Offsetof(credential{}.blob) - 40)
	_ = uint64(72 - unsafe.Offsetof(credential{}.userName))
	_ = uint64(unsafe.Offsetof(credential{}.userName) - 72)
)

func (Store) Put(token string) error {
	if err := rejectEmpty(token); err != nil {
		return err
	}
	if len(token) > MaxCredentialBlob {
		return account.ErrRefreshTokenTooLarge
	}
	secret := []byte(token)
	target, user, err := names()
	if err != nil {
		return err
	}
	item := credential{
		typ:        credTypeGeneric,
		targetName: target,
		blobSize:   uint32(len(secret)),
		blob:       &secret[0],
		persist:    credPersistLocalMachine,
		userName:   user,
	}
	r, _, callErr := procCredWrite.Call(uintptr(unsafe.Pointer(&item)), 0)
	runtime.KeepAlive(secret)
	runtime.KeepAlive(target)
	runtime.KeepAlive(user)
	if r == 0 {
		return account.Wrap(account.CodeStore, "credential_write", callErr)
	}
	return nil
}

func (Store) Get() (string, error) {
	target, _, err := names()
	if err != nil {
		return "", err
	}
	var out *credential
	r, _, callErr := procCredRead.Call(uintptr(unsafe.Pointer(target)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		if errno, ok := callErr.(syscall.Errno); ok && errno == errorNotFound {
			return "", account.ErrNotFound
		}
		return "", account.Wrap(account.CodeStore, "credential_read", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(out)))
	if out == nil || out.blob == nil || out.blobSize == 0 {
		return "", account.ErrNotFound
	}
	buf := unsafe.Slice(out.blob, out.blobSize)
	return string(append([]byte(nil), buf...)), nil
}

func (Store) Delete() error {
	target, _, err := names()
	if err != nil {
		return err
	}
	r, _, callErr := procCredDelete.Call(uintptr(unsafe.Pointer(target)), credTypeGeneric, 0)
	if r == 0 {
		if errno, ok := callErr.(syscall.Errno); ok && errno == errorNotFound {
			return nil
		}
		return account.Wrap(account.CodeStore, "credential_delete", callErr)
	}
	return nil
}

func names() (target, user *uint16, err error) {
	target, err = syscall.UTF16PtrFromString(Target())
	if err != nil {
		return nil, nil, err
	}
	user, err = syscall.UTF16PtrFromString("arion")
	if err != nil {
		return nil, nil, err
	}
	return target, user, nil
}
