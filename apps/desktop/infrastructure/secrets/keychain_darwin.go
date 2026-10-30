//go:build darwin

// Package secrets keeps Arion's local encryption key in the OS credential
// store: the login Keychain on macOS, DPAPI on Windows (PRV-04).
package secrets

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Security -framework Foundation
#include <stdlib.h>
#include "keychain_darwin.h"
*/
import "C"

import (
	"crypto/rand"
	"fmt"
	"unsafe"
)

const (
	errItemNotFound = -25300
	keyBytes        = 32
)

// Key returns the named 256-bit key, creating it on first use. A Keychain
// failure is an error; Arion never falls back to a key on disk.
func Key(service, account string) ([]byte, error) {
	return loadKey(service, account, true)
}

// ExistingKey never creates a replacement for an encrypted store's lost key.
func ExistingKey(service, account string) ([]byte, error) { return loadKey(service, account, false) }

func loadKey(service, account string, create bool) ([]byte, error) {
	cs, ca := C.CString(service), C.CString(account)
	defer C.free(unsafe.Pointer(cs))
	defer C.free(unsafe.Pointer(ca))
	key := make([]byte, keyBytes)
	status := C.arion_keychain_read(cs, ca, (*C.uchar)(unsafe.Pointer(&key[0])), C.int(keyBytes))
	switch status {
	case 0:
		return key, nil
	case errItemNotFound:
		if !create {
			return nil, fmt.Errorf("encryption key missing (%d)", int(status))
		}
	default:
		return nil, fmt.Errorf("keychain read failed (%d)", int(status))
	}
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if status := C.arion_keychain_write(cs, ca, (*C.uchar)(unsafe.Pointer(&key[0])), C.int(keyBytes)); status != 0 {
		return nil, fmt.Errorf("keychain write failed (%d)", int(status))
	}
	return key, nil
}
