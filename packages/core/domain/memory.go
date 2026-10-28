package domain

import (
	"sort"
	"time"
)

// MemoryOrigin records how a vocabulary entry was approved (MEM-01).
type MemoryOrigin string

const (
	// OriginUser is an entry the user typed in Settings.
	OriginUser MemoryOrigin = "user_approved"
	// OriginEdit is an entry learned from the user's correction of inserted
	// dictation; it applies at once and can be undone (ADR 014).
	OriginEdit MemoryOrigin = "learned_edit"
)

const (
	// MaxMemoryEntries bounds the vocabulary. Learned entries beyond it are
	// evicted, least used first; entries the user typed never are.
	MaxMemoryEntries = 2000
	// LearnedTTL removes a learned entry that was neither recalled nor
	// reinforced for this long, as Doubao expires its modify pairs.
	LearnedTTL = 90 * 24 * time.Hour
)

// MemoryEntry is one vocabulary record with its local provenance and
// usage counts. Counts and times never leave the device unencrypted.
type MemoryEntry struct {
	ID     string
	From   string
	To     string
	Origin MemoryOrigin
	// Hotword preserves canonical ASR eligibility when a learned pair is approved.
	Hotword bool
	// Uses counts the edits that taught this entry; Recalls counts the
	// dictations it corrected.
	Uses    int
	Recalls int
	Created time.Time
	// Used is the last time the entry was taught or recalled.
	Used time.Time
}

// LearnOutcome is a content-free result of learning one pair.
type LearnOutcome string

const (
	LearnAdded      LearnOutcome = "added"
	LearnReinforced LearnOutcome = "reinforced"
	LearnReplaced   LearnOutcome = "replaced"
	// LearnReverted removes a learned entry the user corrected back: its
	// replacement was wrong (Doubao suppresses such A→B→A pairs).
	LearnReverted LearnOutcome = "reverted"
	// LearnKept leaves an entry the user typed untouched.
	LearnKept LearnOutcome = "kept"
)

// LearnResult names the entry the outcome concerns.
type LearnResult struct {
	Outcome LearnOutcome
	Entry   MemoryEntry
}

// Learn applies one learned pair to the entries and returns the new list,
// which shares no backing array with the input.
func Learn(entries []MemoryEntry, pair Replacement, id string, now time.Time) ([]MemoryEntry, LearnResult) {
	out := append([]MemoryEntry(nil), entries...)
	for i, e := range out {
		switch {
		case sameText(e.From, pair.From) && e.To == pair.To:
			out[i].Uses++
			out[i].Used = now
			return out, LearnResult{Outcome: LearnReinforced, Entry: out[i]}
		case sameText(e.From, pair.To) && e.To == pair.From:
			if e.Origin != OriginEdit {
				return out, LearnResult{Outcome: LearnKept, Entry: e}
			}
			return append(out[:i], out[i+1:]...), LearnResult{Outcome: LearnReverted, Entry: e}
		case sameText(e.From, pair.From):
			if e.Origin != OriginEdit {
				return out, LearnResult{Outcome: LearnKept, Entry: e}
			}
			out[i].To, out[i].Uses, out[i].Used = pair.To, 1, now
			return out, LearnResult{Outcome: LearnReplaced, Entry: out[i]}
		}
	}
	entry := MemoryEntry{ID: id, From: pair.From, To: pair.To, Origin: OriginEdit, Hotword: true, Uses: 1, Created: now, Used: now}
	return append(out, entry), LearnResult{Outcome: LearnAdded, Entry: entry}
}

// Recall counts one dictation corrected by each fired entry.
func Recall(entries []MemoryEntry, fired []Replacement, now time.Time) []MemoryEntry {
	out := append([]MemoryEntry(nil), entries...)
	for _, f := range fired {
		for i := range out {
			if sameText(out[i].From, f.From) && out[i].To == f.To {
				out[i].Recalls++
				out[i].Used = now
				break
			}
		}
	}
	return out
}

// PruneMemory drops learned entries unused for LearnedTTL and then evicts
// the least used learned entries beyond MaxMemoryEntries. Entries the user
// typed are always kept.
func PruneMemory(entries []MemoryEntry, now time.Time) []MemoryEntry {
	out := make([]MemoryEntry, 0, len(entries))
	for _, e := range entries {
		if e.Origin == OriginEdit && now.Sub(e.Used) > LearnedTTL {
			continue
		}
		out = append(out, e)
	}
	excess := len(out) - MaxMemoryEntries
	if excess <= 0 {
		return out
	}
	var learned []int
	for i, e := range out {
		if e.Origin == OriginEdit {
			learned = append(learned, i)
		}
	}
	sort.SliceStable(learned, func(x, y int) bool {
		a, b := out[learned[x]], out[learned[y]]
		if a.Uses+a.Recalls != b.Uses+b.Recalls {
			return a.Uses+a.Recalls < b.Uses+b.Recalls
		}
		return a.Used.Before(b.Used)
	})
	drop := map[int]bool{}
	for _, i := range learned[:min(excess, len(learned))] {
		drop[i] = true
	}
	kept := out[:0]
	for i, e := range out {
		if !drop[i] {
			kept = append(kept, e)
		}
	}
	return kept
}

// Replacements returns the entries as replacements, in stored order.
func Replacements(entries []MemoryEntry) []Replacement {
	out := make([]Replacement, 0, len(entries))
	for _, e := range entries {
		if e.From != "" {
			out = append(out, Replacement{From: e.From, To: e.To})
		}
	}
	return out
}

// sameText uses the same source equivalence as replacement and recall.
func sameText(a, b string) bool { return SameVocabularySource(a, b) }
