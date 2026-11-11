package ui

import "testing"

func TestShortcutRepeated(t *testing.T) {
	steps, toggles := 0, 0
	var seen []bool
	tt := NewTester(func(c *Context) {
		if c.Shortcut(0, KeyUp) {
			steps++
			seen = append(seen, c.ShortcutRepeated())
		}
		if c.Shortcut(0, KeyT) && !c.ShortcutRepeated() {
			toggles++
		}
		Text(c, "view")
	}, 100, 50)
	tt.KeyDown(KeyUp)
	tt.KeyRepeat(0, KeyUp)
	tt.KeyRepeat(0, KeyUp)
	tt.KeyUp(KeyUp)
	if steps != 3 || len(seen) != 3 || seen[0] || !seen[1] || !seen[2] {
		t.Fatalf("steps %d, repeats %v; want 3, [false true true]", steps, seen)
	}
	tt.KeyDown(KeyT)
	tt.KeyRepeat(0, KeyT)
	tt.KeyRepeat(0, KeyT)
	tt.KeyUp(KeyT)
	tt.Key(0, KeyT)
	if toggles != 2 {
		t.Fatalf("toggled %d times, want once a press: 2", toggles)
	}
	tt.Frame()
	if tt.rt.shortcutRepeat {
		t.Fatal("a frame no shortcut took a key in still tells a repeat")
	}
}

func TestShortcutRepeatedInActions(t *testing.T) {
	var window, element []bool
	tt := NewTester(func(c *Context) {
		c.OnShortcut(0, KeyDown, func() { window = append(window, c.ShortcutRepeated()) })
		ButtonBase(c).Size(40, 20).Label("list").Focus().
			OnShortcut(0, KeyRight, func() { element = append(element, c.ShortcutRepeated()) })
	}, 100, 50)
	tt.KeyDown(KeyDown)
	tt.KeyRepeat(0, KeyDown)
	tt.KeyRepeat(0, KeyRight)
	tt.Key(0, KeyRight)
	if len(window) != 2 || window[0] || !window[1] {
		t.Fatalf("the window's action saw repeats %v, want [false true]", window)
	}
	if len(element) != 2 || !element[0] || element[1] {
		t.Fatalf("the element's action saw repeats %v, want [true false]", element)
	}
}

func TestOverlayShortcutRepeated(t *testing.T) {
	var seen []bool
	tt := NewTester(func(c *Context) {
		Overlay(c, func() {
			if Box(c).Size(10, 10).OverlayShortcut(0, KeyEscape) {
				seen = append(seen, c.ShortcutRepeated())
			}
		})
	}, 100, 50)
	tt.KeyRepeat(0, KeyEscape)
	if len(seen) != 1 || !seen[0] {
		t.Fatalf("the overlay saw repeats %v, want [true]", seen)
	}
}

// hint is a component that listens for its own key wherever it is built.
func hint(c *Context, key Key, ran *int) {
	c.OnWindowShortcut(Cmd, key, func() { *ran++ })
	Text(c, "⌘K")
}

func TestOnWindowShortcutFromDeepInTheTree(t *testing.T) {
	deep, scoped := 0, 0
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			Column(c).Children(func() {
				hint(c, KeyK, &deep)
				c.OnShortcut(Cmd, KeyJ, func() { scoped++ })
			})
		})
	}, 100, 50)
	tt.Key(Cmd, KeyK)
	if deep != 1 {
		t.Fatalf("a window shortcut built deep in the tree ran %d times with nothing focused", deep)
	}
	tt.Key(Cmd, KeyJ)
	if scoped != 0 {
		t.Fatal("OnShortcut deep in the tree heard a key while its parent had no focus")
	}
}

func TestOnWindowShortcutYieldsToTheFocusAndDialogs(t *testing.T) {
	behind, inside, focused := 0, 0, 0
	open := false
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			Column(c).Children(func() {
				hint(c, KeyK, &behind)
			})
			ButtonBase(c).Size(40, 20).Label("field").OnShortcut(Cmd, KeyL, func() { focused++ })
		})
		c.OnWindowShortcut(Cmd, KeyL, func() { t.Error("the window took a key the focused element takes") })
		if open {
			Overlay(c, func() {
				Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Modal().Children(func() {
					Column(c).Children(func() {
						hint(c, KeyI, &inside)
						ButtonBase(c).Size(40, 20).Label("ok")
					})
				})
			})
		}
	}, 200, 100)
	if err := tt.Click("field"); err != nil {
		t.Fatal(err)
	}
	tt.Key(Cmd, KeyL)
	if focused != 1 {
		t.Fatalf("the focused element took Cmd+L %d times", focused)
	}
	open = true
	tt.Frame()
	tt.Frame()
	tt.Key(Cmd, KeyK)
	if behind != 0 {
		t.Fatal("a window shortcut built behind a dialog ran")
	}
	tt.Key(Cmd, KeyI)
	if inside != 1 {
		t.Fatalf("a window shortcut built in the dialog ran %d times", inside)
	}
	open = false
	tt.Frame()
	tt.Frame()
	tt.Key(Cmd, KeyK)
	if behind != 1 {
		t.Fatalf("a window shortcut ran %d times once the dialog closed", behind)
	}
}
