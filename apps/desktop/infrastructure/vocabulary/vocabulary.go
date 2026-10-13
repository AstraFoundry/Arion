// Package vocabulary stores approved replacements encrypted at rest with
// AES-256-GCM; the key lives in the OS credential store (PRV-04, MEM-01).
package vocabulary

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Entry is one approved replacement with local provenance.
type Entry struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Approved bool   `json:"approved"`
	Origin   string `json:"origin"`
}

// ErrInvalid rejects incomplete entries.
var ErrInvalid = errors.New("replacement needs both texts")

const associated = "arion-vocabulary-v1"

// Store is an encrypted file of entries.
type Store struct {
	path string
	aead cipher.AEAD
	mu   sync.Mutex
}

func Open(path string, key []byte) (*Store, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Store{path: path, aead: aead}, nil
}

// List decrypts all entries. A wrong key or tampered file fails closed.
func (s *Store) List() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

func (s *Store) read() ([]Entry, error) {
	sealed, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(sealed) < s.aead.NonceSize() {
		return nil, errors.New("vocabulary file is truncated")
	}
	nonce, body := sealed[:s.aead.NonceSize()], sealed[s.aead.NonceSize():]
	plain, err := s.aead.Open(nil, nonce, body, []byte(associated))
	if err != nil {
		return nil, errors.New("vocabulary file failed authentication")
	}
	var entries []Entry
	if err := json.Unmarshal(plain, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) write(entries []Entry) error {
	plain, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	sealed := s.aead.Seal(nonce, nonce, plain, []byte(associated))
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, sealed, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Add approves a user-entered replacement.
func (s *Store) Add(from, to string) (Entry, error) {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" || to == "" {
		return Entry{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.read()
	if err != nil {
		return Entry{}, err
	}
	id := make([]byte, 8)
	_, _ = rand.Read(id)
	entry := Entry{ID: hex.EncodeToString(id), From: from, To: to, Approved: true, Origin: "user_approved"}
	return entry, s.write(append(entries, entry))
}

// Remove deletes an entry by ID.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.read()
	if err != nil {
		return err
	}
	kept := entries[:0]
	for _, entry := range entries {
		if entry.ID != id {
			kept = append(kept, entry)
		}
	}
	return s.write(kept)
}
