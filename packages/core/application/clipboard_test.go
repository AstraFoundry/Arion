package application

import (
	"errors"
	"testing"
	"time"

	"arion.local/core/domain"
)

type clipRepo struct {
	items            []domain.Clip
	loadErr, saveErr error
	saves            int
}

func (r *clipRepo) Load() ([]domain.Clip, error) { return r.items, r.loadErr }
func (r *clipRepo) Save(items []domain.Clip) error {
	r.saves++
	if r.saveErr != nil {
		return r.saveErr
	}
	r.items = items
	return nil
}

func TestClipboardReadAndWriteFailureDoNotPublishChanges(t *testing.T) {
	r := &clipRepo{}
	s := NewClipboard(r)
	l := domain.DefaultClipLimits()
	now := time.Now()
	items, err := s.Capture(domain.ClipTextPayload("original"), l, now)
	if err != nil {
		t.Fatal(err)
	}
	r.loadErr = errors.New("authentication failed")
	if _, err = s.Refresh(l); err == nil || r.saves != 1 || len(r.items) != 1 {
		t.Fatal("failed read overwritten")
	}
	r.loadErr = nil
	r.saveErr = errors.New("disk full")
	if _, err = s.Pin(items[0].ID, true, l); err == nil || r.items[0].Pinned {
		t.Fatal("failed pin mutated snapshot")
	}
	r.saveErr = nil
	if _, err = s.Pin(items[0].ID, true, l); err != nil {
		t.Fatal(err)
	}
	if kept, err := s.Refresh(l); err != nil || len(kept) != 1 {
		t.Fatal("retention removed a pin")
	}
	if _, err := s.Pin(items[0].ID, false, l); err != nil || len(r.items) != 1 {
		t.Fatal("unpinning removed a record below the count limit")
	}
}

func TestClipboardShortcutNormalization(t *testing.T) {
	for _, c := range []struct {
		os, raw string
		want    HoldKey
	}{{"darwin", "CmdOrCtrl+Shift+V", "shift+super+v"}, {"windows", "CmdOrCtrl+Shift+V", "ctrl+shift+v"}, {"darwin", "Alt+F8", "alt+f8"}} {
		got, ok := ClipboardShortcutKey(c.os, c.raw)
		if !ok || got != c.want {
			t.Fatalf("%s: %q %v", c.raw, got, ok)
		}
	}
	for _, raw := range []string{"v", "Ctrl+Ctrl+V", "Unknown+V", "Escape", ""} {
		if _, ok := ClipboardShortcutKey("darwin", raw); ok {
			t.Fatalf("accepted %q", raw)
		}
	}
}
