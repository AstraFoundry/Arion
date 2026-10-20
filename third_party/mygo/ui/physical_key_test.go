package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestPhysicalKeyMatchesShortcutOnOtherLayout(t *testing.T) {
	hits := 0
	tt := coreNewTester(func(c *context) {
		if c.Shortcut(Super, KeyS) {
			hits++
		}
	}, 200, 100)
	// Cmd+S on a Cyrillic layout: the layout types no US key.
	tt.KeyAt(Super, KeyS)
	if hits != 1 {
		t.Fatalf("physical Super+S matched %d times, want 1", hits)
	}
	// Without a command modifier the physical key is ignored.
	tt.KeyAt(0, KeyS)
	tt.KeyAt(Shift, KeyS)
	if hits != 1 {
		t.Fatalf("unmodified physical S matched a shortcut: %d", hits)
	}
}

func TestPhysicalPunctuationShortcutOnOtherLayout(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  Key
		mods Modifiers
	}{
		{"shifted exclamation at 1", Key1, Ctrl | Shift},
		{"shifted plus at equal", KeyEqual, Ctrl | Shift},
		{"German sharp S at minus", KeyMinus, Ctrl},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hits := 0
			tt := coreNewTester(func(c *context) {
				if c.Shortcut(tc.mods, tc.key) {
					hits++
				}
			}, 200, 100)
			tt.KeyAt(tc.mods, tc.key)
			if hits != 1 {
				t.Fatalf("physical %v at %v matched %d times, want 1", tc.mods, tc.key, hits)
			}
		})
	}
}

func TestPhysicalKeyKeepsCyrillicTyping(t *testing.T) {
	text := "x"
	tt := coreNewTester(func(c *context) {
		coreTextInput(c, &text)
	}, 300, 100)
	tt.Key(0, KeyTab)
	tt.Key(0, KeyEnd)
	// A Cyrillic "ы" on the S key: KeyUnknown, physical S, text delivered.
	tt.send(platformKeyEvent(KeyUnknown, KeyS))
	tt.Type("ы")
	if text != "xы" {
		t.Fatalf("text = %q, want %q", text, "xы")
	}
}

func TestPhysicalKeyGeorgianShortcutAndTyping(t *testing.T) {
	text, hits := "x", 0
	tt := coreNewTester(func(c *context) {
		if c.Shortcut(Super, KeyS) {
			hits++
		}
		coreTextInput(c, &text)
	}, 300, 100)
	tt.Key(0, KeyTab)
	tt.Key(0, KeyEnd)
	// Georgian ს is on the physical S key, but has no US layout key.
	tt.KeyAt(Super, KeyS)
	if hits != 1 {
		t.Fatalf("physical Super+S on Georgian layout matched %d times, want 1", hits)
	}
	tt.send(platformKeyEvent(KeyUnknown, KeyS))
	tt.Type("ს")
	tt.send(platform.SurfaceEvent{Kind: platform.KeyReleased, PhysicalKey: platform.KeyS})
	if text != "xს" || hits != 1 {
		t.Fatalf("Georgian typing: text = %q, shortcut hits = %d; want %q and 1", text, hits, "xს")
	}
}

func TestPhysicalKeyJapaneseComposition(t *testing.T) {
	text, composing, hits := "x", false, 0
	tt := coreNewTester(func(c *context) {
		if c.Shortcut(Super, KeyS) {
			hits++
		}
		in := coreTextInputBase(c, &text).Width(100).Label("name")
		composing = in.Composing()
	}, 300, 100)
	tt.Click("name")
	tt.Key(0, KeyEnd)
	tt.send(platformKeyEvent(KeyUnknown, KeyS))
	tt.Compose("に", 1)
	if !composing || text != "x" || hits != 0 {
		t.Fatalf("during Japanese composition: composing = %v, text = %q, shortcuts = %d", composing, text, hits)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.KeyReleased, PhysicalKey: platform.KeyS})
	tt.Type("日本語")
	if composing || text != "x日本語" || hits != 0 {
		t.Fatalf("after Japanese commit: composing = %v, text = %q, shortcuts = %d", composing, text, hits)
	}
}

func platformKeyEvent(key, physical Key) platform.SurfaceEvent {
	return platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(key), PhysicalKey: platform.Key(physical)}
}

func TestPhysicalKeyReleaseMatchesItsPress(t *testing.T) {
	type keyEv struct {
		kind InputKind
		key  Key
	}
	var got []keyEv
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 100).Focusable().Label("Terminal").AutoFocus().HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputKeyDown || ev.Kind == InputKeyUp {
				got = append(got, keyEv{ev.Kind, ev.Key})
			}
			return true
		})
	}, 200, 100)
	press := func(kind platform.SurfaceEventKind, mods platform.Modifiers) {
		tt.send(platform.SurfaceEvent{Kind: kind, PhysicalKey: platform.KeyS, Mods: mods})
	}
	// Ctrl+ы, then Ctrl is let go of first: the release still stands for S.
	press(platform.KeyPressed, platform.ModCtrl)
	tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, PhysicalKey: platform.KeyS, Repeat: true})
	press(platform.KeyReleased, 0)
	// Plain typing of ы: the release is not turned into S.
	press(platform.KeyPressed, 0)
	press(platform.KeyReleased, 0)
	want := []keyEv{{InputKeyDown, KeyS}, {InputKeyDown, KeyUnknown}, {InputKeyUp, KeyS}, {InputKeyDown, KeyUnknown}, {InputKeyUp, KeyUnknown}}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}

func TestPhysicalKeyReleaseAfterInterruptedPress(t *testing.T) {
	type keyEv struct {
		kind InputKind
		key  Key
	}
	for _, tc := range []struct {
		name     string
		blur     bool
		newPress bool
	}{
		{"blur and refocus before new press", true, true},
		{"missing release then new press", false, true},
		{"blur and refocus before late release", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []keyEv
			tt := coreNewTester(func(c *context) {
				coreBox(c).Size(100, 100).Focusable().Label("Terminal").AutoFocus().HandleInput(func(ev InputEvent) bool {
					if ev.Kind == InputKeyDown || ev.Kind == InputKeyUp {
						got = append(got, keyEv{ev.Kind, ev.Key})
					}
					return true
				})
			}, 200, 100)
			tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, PhysicalKey: platform.KeyS, Mods: platform.ModCtrl})
			if tc.blur {
				tt.SetFocused(false)
				tt.SetFocused(true)
			}
			if tc.newPress {
				tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, PhysicalKey: platform.KeyS})
			}
			tt.send(platform.SurfaceEvent{Kind: platform.KeyReleased, PhysicalKey: platform.KeyS})
			want := []keyEv{{InputKeyDown, KeyS}, {InputKeyUp, KeyUnknown}}
			if tc.newPress {
				want = []keyEv{{InputKeyDown, KeyS}, {InputKeyDown, KeyUnknown}, {InputKeyUp, KeyUnknown}}
			}
			if len(got) != len(want) {
				t.Fatalf("events = %v, want %v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("events = %v, want %v", got, want)
				}
			}
		})
	}
}
