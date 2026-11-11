package ui

import (
	"fmt"
	"slices"
	"testing"
)

// keyNames names the keys the tests press.
var keyNames = map[Key]string{KeyB: "B", KeyS: "S", KeyX: "X", KeyLeft: "Left", KeyTab: "Tab"}

// observed formats what an ObserveInput saw, as "down A", "text a".
func observed(log *[]string, name string) func(InputEvent) {
	return func(ev InputEvent) {
		var s string
		switch ev.Kind {
		case InputKeyDown:
			s = "down " + keyNames[ev.Key]
			if ev.Repeat {
				s += " repeat"
			}
		case InputKeyUp:
			s = "up " + keyNames[ev.Key]
		case InputText:
			s = "text " + ev.Text
		case InputCompose:
			s = "compose " + ev.Text
		case InputCommand:
			s = "command " + ev.Text
		case InputPointerDown:
			s = fmt.Sprintf("press %.0f,%.0f", ev.X, ev.Y)
		case InputPointerUp:
			s = fmt.Sprintf("release %.0f,%.0f", ev.X, ev.Y)
		default:
			s = fmt.Sprint("kind ", ev.Kind)
		}
		if name != "" {
			s = name + " " + s
		}
		*log = append(*log, s)
	}
}

func TestObserveInputSeesKeysTypedInADescendant(t *testing.T) {
	var text string
	var region, sibling []string
	saves := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			r := coreColumn(c).ObserveInput(observed(&region, ""))
			r.Children(func() {
				coreTextInput(c, &text).Label("Field")
			})
			if r.Shortcut(Cmd, KeyS) {
				saves++
			}
			coreColumn(c).ObserveInput(observed(&sibling, "")).Children(func() {
				coreButton(c, "Other")
			})
		})
	}, 300, 200)

	// Nothing has the focus: nothing is observed.
	tt.TypeKey(0, KeyA, "a")
	if len(region) != 0 || len(sibling) != 0 {
		t.Fatalf("without the focus inside: region %q, sibling %q", region, sibling)
	}

	if err := tt.Click("Field"); err != nil {
		t.Fatal(err)
	}
	region = region[:0] // the click on the field
	tt.TypeKey(0, KeyB, "b")
	if text != "b" {
		t.Fatalf("text = %q, want the key typed into the field", text)
	}
	if want := []string{"down B", "text b", "up B"}; !slices.Equal(region, want) {
		t.Fatalf("region saw %q, want %q", region, want)
	}

	// A key a Shortcut around the focus takes is seen, and still taken.
	region = region[:0]
	tt.Key(Cmd, KeyS)
	if saves != 1 {
		t.Fatalf("shortcut ran %d times, want 1", saves)
	}
	if want := []string{"down S", "up S"}; !slices.Equal(region, want) {
		t.Fatalf("region saw %q around the shortcut, want %q", region, want)
	}

	// Repeats, compositions and edit commands come as HandleInput has them.
	region = region[:0]
	tt.KeyRepeat(0, KeyLeft)
	tt.Compose("ni", 2)
	tt.Compose("", 0)
	tt.Command("selectAll")
	if want := []string{"down Left repeat", "compose ni", "compose ", "command selectAll"}; !slices.Equal(region, want) {
		t.Fatalf("region saw %q, want %q", region, want)
	}
	if len(sibling) != 0 {
		t.Fatalf("the sibling region saw %q", sibling)
	}

	// Tab is seen as it leaves; its release comes where the focus went, as
	// with UI Events.
	region = region[:0]
	tt.KeyDown(KeyTab)
	tt.KeyUp(KeyTab)
	if want := []string{"down Tab"}; !slices.Equal(region, want) {
		t.Fatalf("region saw %q, want %q", region, want)
	}
	if want := []string{"up Tab"}; !slices.Equal(sibling, want) {
		t.Fatalf("sibling saw %q once the focus moved to it, want %q", sibling, want)
	}
}

func TestObserveInputRunsOutermostFirstBeforeTheFocus(t *testing.T) {
	var log []string
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().ObserveInput(observed(&log, "outer")).Children(func() {
			coreColumn(c).Padding(10).ObserveInput(observed(&log, "inner")).Children(func() {
				coreBox(c).Size(50, 50).Focusable().Label("Pad").HandleInput(func(ev InputEvent) bool {
					if ev.Kind != InputKeyDown {
						return false
					}
					log = append(log, "handler down "+keyNames[ev.Key])
					return true
				})
			})
		})
	}, 300, 200)
	tt.Key(0, KeyTab)
	log = log[:0]
	tt.KeyDown(KeyX)
	if want := []string{"outer down X", "inner down X", "handler down X"}; !slices.Equal(log, want) {
		t.Fatalf("saw %q, want %q", log, want)
	}
}

func TestObserveInputSeesPressesInside(t *testing.T) {
	var region []string
	tt := coreNewTester(func(c *context) {
		coreRow(c).Fill().AlignItems(Start).Children(func() {
			coreBox(c).Size(100, 100).ObserveInput(observed(&region, "")).Children(func() {
				coreButton(c, "Inside")
			})
			coreButton(c, "Outside")
		})
	}, 300, 200)
	tt.Press(20, 30)
	tt.Release(20, 30)
	if want := []string{"press 20,30", "release 20,30"}; !slices.Equal(region, want) {
		t.Fatalf("region saw %q, want %q", region, want)
	}
	region = region[:0]
	if err := tt.Click("Outside"); err != nil {
		t.Fatal(err)
	}
	if len(region) != 0 {
		t.Fatalf("a press outside was seen: %q", region)
	}
}

func TestObserveInputUnusedKeepsNoChain(t *testing.T) {
	var text string
	tt := coreNewTester(func(c *context) {
		coreTextInput(c, &text).Label("Field")
	}, 300, 100)
	if err := tt.Click("Field"); err != nil {
		t.Fatal(err)
	}
	tt.TypeKey(0, KeyA, "a")
	if tt.rt.observers != 0 || tt.rt.observeBuf != nil {
		t.Fatalf("no element observes, yet observers = %d, chain %v", tt.rt.observers, tt.rt.observeBuf)
	}
}
