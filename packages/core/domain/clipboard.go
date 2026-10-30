package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type ClipKind string

const (
	ClipText  ClipKind = "text"
	ClipLink  ClipKind = "link"
	ClipImage ClipKind = "image"
	ClipFiles ClipKind = "files"
)

var (
	ErrClipInvalid  = errors.New("invalid clipboard payload")
	ErrClipOversize = errors.New("clipboard item exceeds limit")
	ErrClipCapacity = errors.New("clipboard pins exceed capacity")
	ErrClipMissing  = errors.New("clipboard item not found")
)

// ClipPayload contains values only. Files are ordered absolute references;
// adapters validate platform paths and availability before delivery.
type ClipPayload struct {
	Kind  ClipKind `json:"kind"`
	Text  string   `json:"text,omitempty"`
	PNG   []byte   `json:"png,omitempty"`
	Files []string `json:"files,omitempty"`
}

func (p ClipPayload) Validate() error {
	switch p.Kind {
	case ClipText, ClipLink:
		if p.Text == "" || !utf8.ValidString(p.Text) || strings.ContainsRune(p.Text, 0) || len(p.PNG) != 0 || len(p.Files) != 0 {
			return ErrClipInvalid
		}
	case ClipImage:
		if len(p.PNG) < 8 || string(p.PNG[:8]) != "\x89PNG\r\n\x1a\n" || p.Text != "" || len(p.Files) != 0 {
			return ErrClipInvalid
		}
	case ClipFiles:
		if len(p.Files) == 0 || p.Text != "" || len(p.PNG) != 0 {
			return ErrClipInvalid
		}
		for _, path := range p.Files {
			if path == "" || !utf8.ValidString(path) || strings.ContainsRune(path, 0) {
				return ErrClipInvalid
			}
		}
	default:
		return ErrClipInvalid
	}
	return nil
}

func ClipTextPayload(text string) ClipPayload {
	return ClipPayload{Kind: ClipText, Text: text}
}

func (p ClipPayload) Bytes() int64 {
	n := int64(len(p.Text) + len(p.PNG))
	for _, path := range p.Files {
		n += int64(len(path))
	}
	return n
}

// ClipID is content-derived but is persisted only inside the encrypted snapshot.
func (p ClipPayload) ID() string {
	b, _ := json.Marshal(p)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type Clip struct {
	ID      string      `json:"id"`
	Payload ClipPayload `json:"payload"`
	Copied  time.Time   `json:"copied"`
	Pinned  bool        `json:"pinned,omitempty"`
}

type ClipLimits struct {
	Count     int   `json:"count"`
	Bytes     int64 `json:"bytes"`
	ItemBytes int64 `json:"itemBytes"`
}

func DefaultClipLimits() ClipLimits {
	return ClipLimits{Count: 200, Bytes: 100 << 20, ItemBytes: 10 << 20}
}

func (l ClipLimits) Valid() bool {
	return l.Count >= 1 && l.Count <= 2000 && l.Bytes >= 1<<20 && l.Bytes <= 500<<20 && l.ItemBytes >= 1<<10 && l.ItemBytes <= l.Bytes && l.ItemBytes <= 50<<20
}

func SortClips(items []Clip) {
	slices.SortStableFunc(items, func(a, b Clip) int {
		if a.Pinned != b.Pinned {
			if a.Pinned {
				return -1
			}
			return 1
		}
		if n := b.Copied.Compare(a.Copied); n != 0 {
			return n
		}
		return strings.Compare(a.ID, b.ID)
	})
}

// PruneClips preserves pins and applies count/storage limits to the ordinary history.
// Pinned data cannot be silently evicted to make a policy fit.
func PruneClips(items []Clip, limits ClipLimits) ([]Clip, error) {
	if !limits.Valid() {
		return nil, ErrClipInvalid
	}
	items = slices.Clone(items)
	SortClips(items)
	var pinnedBytes int64
	for _, item := range items {
		if item.Pinned {
			pinnedBytes += item.Payload.Bytes()
		}
	}
	if pinnedBytes > limits.Bytes {
		return nil, ErrClipCapacity
	}
	var out []Clip
	var used int64
	count := 0
	for _, item := range items {
		size := item.Payload.Bytes()
		if !item.Pinned && (count >= limits.Count || used+size > limits.Bytes) {
			continue
		}
		out = append(out, item)
		used += size
		if !item.Pinned {
			count++
		}
	}
	return out, nil
}

func CaptureClip(items []Clip, payload ClipPayload, limits ClipLimits, now time.Time) ([]Clip, error) {
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	if payload.Bytes() > limits.ItemBytes {
		return nil, ErrClipOversize
	}
	id := payload.ID()
	next := slices.Clone(items)
	found := false
	for i := range next {
		if next[i].ID == id {
			next[i].Copied = now
			found = true
			break
		}
	}
	if !found {
		next = append(next, Clip{ID: id, Payload: payload, Copied: now})
	}
	next, err := PruneClips(next, limits)
	if err != nil {
		return nil, err
	}
	for _, item := range next {
		if item.ID == id {
			return next, nil
		}
	}
	return nil, ErrClipCapacity
}

// FindClips filters already-sorted values without mutating the snapshot.
func FindClips(items []Clip, query string, kind ClipKind) []Clip {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []Clip
	for _, item := range items {
		p := item.Payload
		if kind != "" && p.Kind != kind && !(kind == ClipText && p.Kind == ClipLink) {
			continue
		}
		haystack := p.Text + "\n" + strings.Join(p.Files, "\n")
		if query == "" || strings.Contains(strings.ToLower(haystack), query) {
			out = append(out, item)
		}
	}
	return out
}
