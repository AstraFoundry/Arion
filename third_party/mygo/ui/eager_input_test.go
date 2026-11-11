package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestInputReachesLocalCopies: a control's input reaches what it binds as
// its constructor returns, so a local copy compared with the value it
// started from, without asking Changed, sees the user's edit, as with the
// widgets of Dear ImGui and egui.
func TestInputReachesLocalCopies(t *testing.T) {
	var language, style, query string
	var sync bool
	var volume float64
	tt := NewTester(func(c *Context) {
		Column(c).Gap(8).Children(func() {
			chosen := language
			RadioGroup(c, func() {
				Radio(c, &chosen, "", "System")
				Radio(c, &chosen, "de", "Deutsch")
				Radio(c, &chosen, "ja", "日本語")
			}).Label("Language")
			if chosen != language {
				language = chosen
			}
			on := sync
			Checkbox(c, &on, "Sync")
			if on != sync {
				sync = on
			}
			s := style
			Select(c, &s, []string{"Body", "Heading"}).Label("Style")
			if s != style {
				style = s
			}
			v := volume
			Slider(c.Key("volume"), &v, 0, 1).Step(0.25).Label("Volume")
			if v != volume {
				volume = v
			}
			q := query
			TextInput(c.Key("query"), &q).Label("Query")
			if q != query {
				query = q
			}
		})
	}, 400, 400)

	if err := tt.Click("Deutsch"); err != nil {
		t.Fatal(err)
	}
	if language != "de" {
		t.Errorf("after clicking Deutsch, language %q", language)
	}
	// The group's arrow keys move among the radios built in it.
	tt.Key(0, KeyDown)
	if language != "ja" {
		t.Errorf("after the down arrow, language %q", language)
	}
	if err := tt.Click("Sync"); err != nil {
		t.Fatal(err)
	}
	if !sync {
		t.Error("the click did not check Sync")
	}
	if err := tt.Click("Style"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Heading"); err != nil {
		t.Fatal(err)
	}
	if style != "Heading" {
		t.Errorf("after choosing Heading, style %q", style)
	}
	if err := tt.Click("Volume"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, KeyHome)
	tt.Key(0, KeyRight)
	if volume != 0.25 {
		t.Errorf("after Home and the right arrow, volume %v, want a step of 0.25", volume)
	}
	if err := tt.Click("Query"); err != nil {
		t.Fatal(err)
	}
	tt.Type("go")
	if query != "go" {
		t.Errorf("after typing, query %q", query)
	}
}

// TestInputReachesLocalCopiesOfParts: a part's method that builds an
// element, as a select's Item, is a constructor too: the choice is in the
// local copy once the popup is built.
func TestInputReachesLocalCopiesOfParts(t *testing.T) {
	size := "S"
	tt := NewTester(func(c *Context) {
		chosen := size
		s := SelectBase(c, &chosen)
		s.Trigger.Label("Size").Size(80, 24)
		s.Popup(func(panel Element) {
			for _, v := range []string{"S", "M", "L"} {
				// No control is built after the item: its own call
				// takes its input.
				s.Item(v).Label("Size "+v).Size(80, 24)
			}
		})
		if chosen != size {
			size = chosen
		}
	}, 300, 300)
	if err := tt.Click("Size"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Size L"); err != nil {
		t.Fatal(err)
	}
	if size != "L" {
		t.Errorf("after choosing Size L, size %q", size)
	}
}

// TestListChangedInSplit: List is a constructor too, in both its forms, so
// a click is Changed as it returns, also in a split's pane, whose
// constructor is still building (#364).
func TestListChangedInSplit(t *testing.T) {
	rows := []string{"row 0", "row 1", "row 2"}
	for _, place := range []string{"column", "SplitVertical", "Split"} {
		for _, form := range []string{"row", "Rows"} {
			sel := -1
			s := ListState{Selected: &sel}
			changed := 0
			list := func(c *Context) {
				var l Element
				if form == "row" {
					l = List(c, &s, len(rows), func(i int) { Text(c, rows[i]) })
				} else {
					l = List(c, &s, len(rows)).Rows(func(r ListRow) { Text(r.Context, rows[r.Index]) })
				}
				if l.Grow(1).Changed() {
					changed++
				}
			}
			tt := NewTester(func(c *Context) {
				size := float32(60)
				switch place {
				case "column":
					Column(c).Grow(1).Children(func() { list(c) })
				case "SplitVertical":
					SplitVertical(c, &size, func() { list(c) }, func() { Text(c, "other") }).Grow(1)
				case "Split":
					Split(c, &size, func() { Text(c, "other") }, func() { list(c) }).Grow(1)
				}
			}, 300, 300)
			if err := tt.Click("row 1"); err != nil {
				t.Fatal(err)
			}
			if sel != 1 || changed != 1 {
				t.Errorf("%s, %s: chose %d, Changed %d times", place, form, sel, changed)
			}
		}
	}
}

// TestListClickThenTypeInOneFrame clicks a row and types the first letters
// of another before the window draws a frame, as macOS delivers them: the
// click is the rows', queued first, so it applies before the list takes
// the typing, in both of List's forms.
func TestListClickThenTypeInOneFrame(t *testing.T) {
	words := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel"}
	label := func(i int) string { return fmt.Sprintf("%s %d", words[i%len(words)], i) }
	for _, form := range []string{"row", "Rows"} {
		chosen := -1
		s := ListState{Selected: &chosen, Label: label}
		tt := NewTester(func(c *Context) {
			if form == "row" {
				List(c, &s, 100, func(i int) { Text(c, label(i)).Padding(6, 10) }).Fill()
			} else {
				List(c, &s, 100).Fill().Rows(func(r ListRow) { Text(r.Context, label(r.Index)).Padding(6, 10) })
			}
		}, 300, 300)
		r, ok := tt.Find("alpha 0")
		if !ok {
			t.Fatal("no row alpha 0")
		}
		x, y := float64(r.X+r.W/2), float64(r.Y+r.H/2)
		events := []platform.SurfaceEvent{
			{Kind: platform.PointerDown, X: x, Y: y, Clicks: 1},
			{Kind: platform.PointerUp, X: x, Y: y, Clicks: 1},
		}
		for _, k := range []Key{KeyH, KeyO, KeyT} {
			events = append(events,
				platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(k)},
				platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(k)})
		}
		for _, ev := range events {
			tt.rt.event(ev)
		}
		tt.settle()
		if chosen != 7 {
			t.Errorf("%s: clicking alpha 0 and typing hot chose %d, want 7 (hotel 7)", form, chosen)
		}
	}
}
