package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A cheat sheet sees the keys held in the window while its search field
// has the focus and takes them, and while an element that takes every
// key does, without taking them from either; they go up as they are let
// go of, and all at once as the window loses the keyboard.
func TestKeysHeldWhateverHasTheFocus(t *testing.T) {
	query := ""
	var taken []Key
	raw := false
	var held []Key
	tt := coreNewTester(func(c *context) {
		if raw {
			c.keySet, c.nextKey = true, "game"
			coreBox(c).Size(40, 40).Label("game").Focusable().AutoFocus().HandleInput(func(ev InputEvent) bool {
				if ev.Kind == InputKeyDown {
					taken = append(taken, ev.Key)
				}
				return true
			})
		} else {
			coreSearchField(c, &query).Label("Search").Width(160).AutoFocus()
		}
		held = c.KeysDown()
		if c.KeyDown(KeyK) {
			coreText(c, "K lit")
		}
	}, 300, 100)
	if !tt.Focused("Search") {
		t.Fatal("the search field has no focus")
	}
	tt.KeyDown(KeyK)
	if !tt.HasText("K lit") || !slices.Equal(held, []Key{KeyK}) {
		t.Fatalf("K held in the search field: %v", held)
	}
	// The field takes Left; it is held all the same, after K.
	tt.KeyDown(KeyLeft)
	if !slices.Equal(held, []Key{KeyK, KeyLeft}) {
		t.Errorf("K and Left held: %v", held)
	}
	tt.KeyUp(KeyK)
	if tt.HasText("K lit") || !slices.Equal(held, []Key{KeyLeft}) {
		t.Errorf("K let go of: %v", held)
	}
	tt.KeyUp(KeyLeft)
	if held != nil {
		t.Errorf("nothing held: %v", held)
	}

	raw = true
	tt.Frame()
	if !tt.Focused("game") {
		t.Fatal("the game view has no focus")
	}
	tt.KeyDown(KeyW)
	tt.KeyDown(KeyW) // auto-repeat
	if !slices.Equal(held, []Key{KeyW}) || !slices.Equal(taken, []Key{KeyW, KeyW}) {
		t.Errorf("W held, the element taking it: held %v, taken %v", held, taken)
	}
	tt.KeyDown(KeyD)
	tt.SetFocused(false)
	if held != nil {
		t.Errorf("the window losing the keyboard kept %v held", held)
	}
	tt.SetFocused(true)
	tt.KeyUp(KeyW) // let go of while away
	if held != nil {
		t.Errorf("a key let go of with nothing held: %v", held)
	}
}

// A key pressed with Cmd on a layout that types no US key for it, as S on
// a Cyrillic layout, is held as shortcuts take it, by its position, and
// goes up by its position though the release names no key.
func TestKeysHeldByPosition(t *testing.T) {
	var held []Key
	tt := coreNewTester(func(c *context) { held = c.KeysDown() }, 100, 100)
	tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, PhysicalKey: platform.KeyS, Mods: platform.ModCtrl})
	if !slices.Equal(held, []Key{KeyS}) {
		t.Fatalf("Ctrl and the key at S: %v", held)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.KeyReleased, PhysicalKey: platform.KeyS})
	if held != nil {
		t.Errorf("let go of: %v", held)
	}
	// A key typed with no shortcut modifier is held by the key it is,
	// and goes up by its position, whatever the release calls it.
	tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.KeyY, PhysicalKey: platform.KeyZ})
	if !slices.Equal(held, []Key{KeyY}) {
		t.Fatalf("Y at Z on a German layout: %v", held)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.KeyUnknown, PhysicalKey: platform.KeyZ})
	if held != nil {
		t.Errorf("let go of by position: %v", held)
	}
	// A key no shortcut names is not held.
	tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, PhysicalKey: platform.KeyQ})
	if held != nil {
		t.Errorf("a key of no name typed: %v", held)
	}
}

// Keys let go of draw a frame only for a view that read the keys held; a
// key pressed draws one anyway, for its shortcuts.
func TestKeysHeldFrames(t *testing.T) {
	read := false
	tt := newRepaintTester(func(c *context) {
		coreText(c, "Hello")
		if read {
			c.KeyDown(KeyA)
		}
	})
	events := []platform.SurfaceEvent{
		{Kind: platform.KeyPressed, Key: platform.KeyA},
		{Kind: platform.KeyReleased, Key: platform.KeyA},
		{Kind: platform.KeyPressed, Key: platform.KeyB},
		{Kind: platform.SurfaceBlur},
	}
	for _, ev := range events {
		tt.h.requested.Store(false)
		tt.rt.event(ev)
		if ev.Kind != platform.KeyPressed && tt.h.requested.Load() {
			t.Errorf("%v, nothing reading the keys: asked for a frame", ev.Kind)
		}
		tt.frame(0)
	}
	read = true
	tt.rt.changed()
	tt.frame(0)
	tt.frame(0)
	for _, ev := range events {
		tt.h.requested.Store(false)
		tt.rt.event(ev)
		if !tt.h.requested.Load() {
			t.Errorf("%v, the view reading the keys: asked for no frame", ev.Kind)
		}
		tt.frame(0)
	}
}
