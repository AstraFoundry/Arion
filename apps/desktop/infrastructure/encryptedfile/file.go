// Package encryptedfile persists bounded authenticated snapshots. Encryption
// happens before any bytes reach a temporary file or the final destination.
package encryptedfile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type File struct {
	Path       string
	aead       cipher.AEAD
	associated []byte
	MaxBytes   int64
}

func New(path string, key []byte, associated string, maxBytes int64) (*File, error) {
	if len(key) != 32 || associated == "" || maxBytes <= 0 {
		return nil, errors.New("invalid encrypted file configuration")
	}
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	a, err := cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}
	return &File{Path: path, aead: a, associated: []byte(associated), MaxBytes: maxBytes}, nil
}

// Read distinguishes an absent file from corrupt/unreadable data.
func (f *File) Read() ([]byte, error) {
	info, err := os.Lstat(f.Path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > f.MaxBytes {
		return nil, errors.New("invalid encrypted file")
	}
	fd, err := os.Open(f.Path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	b, err := io.ReadAll(io.LimitReader(fd, f.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > f.MaxBytes || len(b) < f.aead.NonceSize()+f.aead.Overhead() {
		return nil, errors.New("encrypted file is truncated or oversized")
	}
	n := f.aead.NonceSize()
	p, err := f.aead.Open(nil, b[:n], b[n:], f.associated)
	if err != nil {
		return nil, errors.New("encrypted file failed authentication")
	}
	return p, nil
}

func (f *File) Write(plain []byte) error {
	if int64(len(plain)+f.aead.NonceSize()+f.aead.Overhead()) > f.MaxBytes {
		return errors.New("encrypted file exceeds limit")
	}
	nonce := make([]byte, f.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	sealed := f.aead.Seal(nonce, nonce, plain, f.associated)
	if err := os.MkdirAll(filepath.Dir(f.Path), 0700); err != nil {
		return err
	}
	if info, err := os.Lstat(f.Path); err == nil && !info.Mode().IsRegular() {
		return errors.New("invalid encrypted destination")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fd, err := os.CreateTemp(filepath.Dir(f.Path), ".sealed-*")
	if err != nil {
		return err
	}
	tmp := fd.Name()
	defer os.Remove(tmp)
	if _, err = fd.Write(sealed); err == nil {
		err = fd.Sync()
	}
	closeErr := fd.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp, f.Path)
}
