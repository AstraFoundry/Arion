//go:build !darwin && !windows

// Package secrets keeps Arion's local encryption key in the OS credential
// store: the login Keychain on macOS, DPAPI on Windows (PRV-04).
// Other hosts have no such store. Keys are not written to disk.
package secrets

import "errors"

// ErrUnsupported means this OS cannot keep a local encryption key.
var ErrUnsupported = errors.New("encryption key store unsupported")

func Key(string, string) ([]byte, error)         { return nil, ErrUnsupported }
func ExistingKey(string, string) ([]byte, error) { return nil, ErrUnsupported }
