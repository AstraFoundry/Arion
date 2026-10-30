// Package vocabulary stores approved replacements encrypted at rest with
// AES-256-GCM; the key lives in the OS credential store (PRV-04, MEM-01).
// Entries learned from the user's edits live in the same file with their
// usage counts (ADR 014). Every change rewrites the file atomically: a
// crash leaves either the old or the new file, never a mix.
package vocabulary

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/encryptedfile"
)

// Entry is a correction pair or a canonical-only hotword with local provenance. Files written
// before learning existed load with zero counts and times.
type Entry struct {
	ID             string `json:"id"`
	From           string `json:"from"`
	To             string `json:"to"`
	Approved       bool   `json:"approved"`
	Hotword        bool   `json:"hotword,omitempty"`
	KeyboardPinyin string `json:"keyboardPinyin,omitempty"`
	Origin         string `json:"origin"`
	Uses           int    `json:"uses,omitempty"`
	Recalls        int    `json:"recalls,omitempty"`
	// Created and Used are Unix seconds.
	Created int64 `json:"created,omitempty"`
	Used    int64 `json:"used,omitempty"`
}

func (e Entry) memory() domain.MemoryEntry {
	origin := domain.MemoryOrigin(e.Origin)
	if origin != domain.OriginEdit {
		origin = domain.OriginUser
	}
	return domain.MemoryEntry{ID: e.ID, From: e.From, To: e.To, Origin: origin, Hotword: e.Hotword || origin == domain.OriginEdit, Uses: e.Uses, Recalls: e.Recalls,
		KeyboardPinyin: e.KeyboardPinyin, Created: time.Unix(e.Created, 0), Used: time.Unix(e.Used, 0)}
}

func entryOf(m domain.MemoryEntry) Entry {
	return Entry{ID: m.ID, From: m.From, To: m.To, Approved: true, Origin: string(m.Origin), Hotword: m.Hotword, Uses: m.Uses, Recalls: m.Recalls,
		KeyboardPinyin: m.KeyboardPinyin, Created: m.Created.Unix(), Used: m.Used.Unix()}
}

// Memory converts entries for the domain's learning rules.
func Memory(entries []Entry) []domain.MemoryEntry {
	out := make([]domain.MemoryEntry, 0, len(entries))
	for _, e := range entries {
		if e.Approved {
			out = append(out, e.memory())
		}
	}
	return out
}

// ErrInvalid rejects incomplete, unsafe or identity entries.
var ErrInvalid = domain.ErrInvalidVocabularyEdit

// ErrConflict rejects a heard form already approved for another written form.
var ErrConflict = errors.New("replacement conflicts with an approved entry")

const associated = "arion-vocabulary-v1"

// Store is an encrypted file of entries.
type Store struct {
	path string
	file *encryptedfile.File
	mu   sync.Mutex
}

func Open(path string, key []byte) (*Store, error) {
	file, err := encryptedfile.New(path, key, associated, 64<<20)
	if err != nil {
		return nil, err
	}
	return &Store{path: path, file: file}, nil
}

// List decrypts all entries. A wrong key or tampered file fails closed.
func (s *Store) List() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

func (s *Store) read() ([]Entry, error) {
	plain, err := s.file.Read()
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
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
	return s.file.Write(plain)
}

func newEntryID() string {
	id := make([]byte, 8)
	_, _ = rand.Read(id)
	return hex.EncodeToString(id)
}

// Learn applies learned pairs in one write, then expires and evicts learned
// entries (domain.PruneMemory). A file that cannot be read is left as it
// is: learning never overwrites memory it failed to decrypt.
func (s *Store) Learn(pairs []domain.Replacement, now time.Time) ([]domain.LearnResult, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	return s.update(func(memory []domain.MemoryEntry) ([]domain.MemoryEntry, []domain.LearnResult) {
		var results []domain.LearnResult
		for _, pair := range pairs {
			var result domain.LearnResult
			memory, result = domain.Learn(memory, pair, newEntryID(), now)
			results = append(results, result)
		}
		return domain.PruneMemory(memory, now), results
	})
}

// Recall counts the dictations the fired entries corrected.
func (s *Store) Recall(fired []domain.Replacement, now time.Time) error {
	if len(fired) == 0 {
		return nil
	}
	_, err := s.update(func(memory []domain.MemoryEntry) ([]domain.MemoryEntry, []domain.LearnResult) {
		return domain.Recall(memory, fired, now), nil
	})
	return err
}

func (s *Store) update(change func([]domain.MemoryEntry) ([]domain.MemoryEntry, []domain.LearnResult)) ([]domain.LearnResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.read()
	if err != nil {
		return nil, err
	}
	// Unapproved entries are not memory; keep them as they are.
	var kept []Entry
	for _, e := range entries {
		if !e.Approved {
			kept = append(kept, e)
		}
	}
	memory, results := change(Memory(entries))
	for _, m := range memory {
		kept = append(kept, entryOf(m))
	}
	return results, s.write(kept)
}

// Add approves a user-entered replacement.
func (s *Store) Add(from, to string) (Entry, error) {
	entries, err := s.AddVariants([]string{from}, to)
	if err != nil {
		return Entry{}, err
	}
	return entries[0], nil
}

// AddVariants approves distinct heard forms for one written form in one
// atomic write. Existing pairs keep their identity and usage history; explicit
// approval promotes learned pairs so they no longer expire or get re-pointed.
func (s *Store) AddVariants(froms []string, to string) ([]Entry, error) {
	pairs, err := domain.NormalizeVocabularyEdit(froms, to)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.read()
	if err != nil {
		return nil, err
	}
	// Check every approved match before changing any entry. Unapproved
	// proposals cannot prevent an explicit approval.
	for _, pair := range pairs {
		for _, entry := range entries {
			if entry.Approved && domain.SameVocabularySource(entry.From, pair.From) && entry.To != pair.To {
				return nil, ErrConflict
			}
		}
	}
	var added []Entry
	changed := false
	now := time.Now().Unix()
	for _, pair := range pairs {
		found := false
		for i, entry := range entries {
			if !entry.Approved || !domain.SameVocabularySource(entry.From, pair.From) {
				continue
			}
			if entry.Origin != string(domain.OriginUser) {
				entries[i].Hotword = entry.Hotword || entry.Origin == string(domain.OriginEdit)
				entries[i].Origin = string(domain.OriginUser)
				changed = true
			}
			if !found {
				added = append(added, entries[i])
				found = true
			}
		}
		if !found {
			entry := Entry{ID: newEntryID(), From: pair.From, To: pair.To, Approved: true, Origin: string(domain.OriginUser), Created: now, Used: now}
			entries = append(entries, entry)
			added = append(added, entry)
			changed = true
		}
	}
	if changed {
		if err := s.write(entries); err != nil {
			return nil, err
		}
	}
	return added, nil
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

// AddHotword saves one independent canonical term in the same encrypted file.
// It does not add a replacement or promote an existing learned alias.
func (s *Store) AddHotword(text string) (Entry, error) {
	term, err := domain.NormalizeHotword(text)
	if err != nil {
		return Entry{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.read()
	if err != nil {
		return Entry{}, err
	}
	for _, entry := range entries {
		if entry.Approved && entry.Hotword && entry.From == "" && entry.To == term {
			return entry, nil
		}
	}
	now := time.Now().Unix()
	entry := Entry{ID: newEntryID(), To: term, Hotword: true, Approved: true, Origin: string(domain.OriginUser), Created: now, Used: now}
	entries = append(entries, entry)
	if err := s.write(entries); err != nil {
		return Entry{}, err
	}
	return entry, nil
}
