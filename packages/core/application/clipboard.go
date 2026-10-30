package application

import (
	"slices"
	"strings"
	"sync"
	"time"

	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type ClipboardPreferences struct {
	Enabled  bool              `json:"enabled"`
	Shortcut string            `json:"shortcut"`
	Limits   domain.ClipLimits `json:"limits"`
}

func DefaultClipboardPreferences() ClipboardPreferences {
	return ClipboardPreferences{Shortcut: "CmdOrCtrl+Shift+V", Limits: domain.DefaultClipLimits()}
}

// ClipboardShortcutKey normalizes accelerator spelling for conflicts with the
// shared dictation listener. Ordinary unmodified typing is never global.
func ClipboardShortcutKey(platform, raw string) (HoldKey, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(raw)), "+")
	if len(parts) == 0 {
		return "", false
	}
	var mods KeyModifiers
	for _, part := range parts[:len(parts)-1] {
		var bit KeyModifiers
		switch part {
		case "cmdorctrl":
			if platform == "darwin" {
				bit = KeySuper
			} else {
				bit = KeyCtrl
			}
		case "ctrl", "control":
			bit = KeyCtrl
		case "cmd", "command", "super", "win":
			bit = KeySuper
		case "alt", "option":
			bit = KeyAlt
		case "shift":
			bit = KeyShift
		default:
			return "", false
		}
		if mods&bit != 0 {
			return "", false
		}
		mods |= bit
	}
	key, ok := HoldShortcut(platform, mods, HoldKey(parts[len(parts)-1]))
	return key, ok && (mods != 0 || isFunctionKey(HoldKey(parts[len(parts)-1])))
}

// Clipboard serializes mutations and publishes only successfully persisted
// snapshots. The composition root calls this service off the UI thread.
type Clipboard struct {
	mu   sync.Mutex
	repo ports.ClipboardRepository
}

func NewClipboard(repo ports.ClipboardRepository) *Clipboard { return &Clipboard{repo: repo} }

func (s *Clipboard) Update(change func([]domain.Clip) ([]domain.Clip, error)) ([]domain.Clip, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.repo.Load()
	if err != nil {
		return nil, err
	}
	next, err := change(slices.Clone(items))
	if err != nil {
		return nil, err
	}
	if !slices.EqualFunc(items, next, func(a, b domain.Clip) bool { return a.ID == b.ID && a.Pinned == b.Pinned && a.Copied.Equal(b.Copied) }) {
		if err := s.repo.Save(next); err != nil {
			return nil, err
		}
	}
	return next, nil
}

func (s *Clipboard) Refresh(limits domain.ClipLimits, now time.Time) ([]domain.Clip, error) {
	return s.Update(func(items []domain.Clip) ([]domain.Clip, error) { return domain.PruneClips(items, limits, now) })
}
func (s *Clipboard) Capture(payload domain.ClipPayload, limits domain.ClipLimits, now time.Time) ([]domain.Clip, error) {
	return s.Update(func(items []domain.Clip) ([]domain.Clip, error) {
		return domain.CaptureClip(items, payload, limits, now)
	})
}
func (s *Clipboard) Pin(id string, pinned bool, limits domain.ClipLimits, now time.Time) ([]domain.Clip, error) {
	return s.Update(func(items []domain.Clip) ([]domain.Clip, error) {
		found := false
		for i := range items {
			if items[i].ID == id {
				items[i].Pinned = pinned
				found = true
			}
		}
		if !found {
			return nil, domain.ErrClipMissing
		}
		return domain.PruneClips(items, limits, now)
	})
}
