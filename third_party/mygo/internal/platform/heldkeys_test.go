package platform

import (
	"slices"
	"testing"
)

// heldBackend sends what a backend would, through HeldKeys, and keeps what
// reached the content.
type heldBackend struct {
	held HeldKeys
	sent []SurfaceEvent
}

func (b *heldBackend) send(ev SurfaceEvent) bool {
	if !b.held.Track(ev) {
		return false
	}
	b.sent = append(b.sent, ev)
	return true
}

func (b *heldBackend) releases() []Key {
	var keys []Key
	for _, ev := range b.sent {
		if ev.Kind == KeyReleased {
			keys = append(keys, ev.Key)
		}
	}
	return keys
}

// An input method taking a key's release lets go of the key by its
// position; when it hands the release back after all, as IBus does, the
// content gets it once.
func TestHeldKeysLift(t *testing.T) {
	var b heldBackend
	b.send(SurfaceEvent{Kind: KeyPressed, Key: KeyA, PhysicalKey: KeyQ}) // A on AZERTY
	b.held.Lift(KeyUnknown, KeyQ, ModShift, b.send)
	if got := b.releases(); !slices.Equal(got, []Key{KeyA}) {
		t.Fatalf("released %v, want A", got)
	}
	if ev := b.sent[len(b.sent)-1]; ev.PhysicalKey != KeyQ || ev.Mods != ModShift {
		t.Errorf("the release: %+v", ev)
	}
	b.send(SurfaceEvent{Kind: KeyReleased, Key: KeyA, PhysicalKey: KeyQ})
	if got := b.releases(); len(got) != 1 {
		t.Errorf("the release handed back reached the content: %v", got)
	}
	// Pressed again, its release comes as usual.
	b.send(SurfaceEvent{Kind: KeyPressed, Key: KeyA, PhysicalKey: KeyQ})
	b.send(SurfaceEvent{Kind: KeyReleased, Key: KeyA, PhysicalKey: KeyQ})
	if got := b.releases(); len(got) != 2 {
		t.Errorf("a later release: %v", got)
	}
	// Nothing held there: nothing to let go of.
	b.held.Lift(KeyUnknown, KeyW, 0, b.send)
	if got := b.releases(); len(got) != 2 {
		t.Errorf("lifted a key not held: %v", got)
	}
	// A release of a key never held goes on, as before.
	b.send(SurfaceEvent{Kind: KeyReleased, Key: KeyZ})
	if got := b.releases(); len(got) != 3 {
		t.Errorf("a release with nothing held: %v", got)
	}
}

// A menu's loop taking the keyboard lets go of every key held, once; a
// repeat holds a key once; the window losing the keyboard forgets them.
func TestHeldKeysLiftAll(t *testing.T) {
	var b heldBackend
	b.send(SurfaceEvent{Kind: KeyPressed, Key: KeyF, PhysicalKey: KeyF, Mods: ModAlt})
	b.send(SurfaceEvent{Kind: KeyPressed, Key: KeyF, PhysicalKey: KeyF, Mods: ModAlt, Repeat: true})
	b.send(SurfaceEvent{Kind: KeyPressed, Key: KeyEnter})
	b.held.LiftAll(0, b.send)
	if got := b.releases(); !slices.Equal(got, []Key{KeyF, KeyEnter}) {
		t.Fatalf("released %v, want F and Enter", got)
	}
	b.held.LiftAll(0, b.send)
	b.send(SurfaceEvent{Kind: KeyReleased, Key: KeyEnter})
	if got := b.releases(); len(got) != 2 {
		t.Errorf("released again: %v", got)
	}
	b.send(SurfaceEvent{Kind: KeyPressed, Key: KeyG, PhysicalKey: KeyG})
	b.send(SurfaceEvent{Kind: SurfaceBlur})
	b.held.LiftAll(0, b.send)
	if got := b.releases(); len(got) != 2 {
		t.Errorf("released after the window lost the keyboard: %v", got)
	}
	// A closed window takes nothing; the keys are let go of all the same.
	b.send(SurfaceEvent{Kind: KeyPressed, Key: KeyH})
	b.held.LiftAll(0, func(SurfaceEvent) bool { return false })
	if len(b.held.held) != 0 {
		t.Errorf("held after LiftAll: %v", b.held.held)
	}
}
