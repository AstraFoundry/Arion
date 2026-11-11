package ui_test

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestCaretWidthRasterizesDIPsForInputAndArea(t *testing.T) {
	for _, area := range []bool{false, true} {
		name := "input"
		if area {
			name = "area"
		}
		t.Run(name, func(t *testing.T) {
			for _, scale := range []struct {
				name  string
				value float32
				index int
			}{{name: "scale 1", value: 1}, {name: "scale 2", value: 2, index: 1}} {
				t.Run(scale.name, func(t *testing.T) {
					for _, test := range []struct {
						name     string
						override bool
						width    float32
						pixels   [2]int
					}{
						{name: "default one DIP", pixels: [2]int{1, 2}},
						{name: "explicit two DIPs", override: true, width: 2, pixels: [2]int{2, 4}},
						{name: "fraction rounds in device pixels", override: true, width: 1.5, pixels: [2]int{2, 3}},
						{name: "positive fraction retains one device pixel", override: true, width: .1, pixels: [2]int{1, 1}},
						{name: "zero hides only caret paint", override: true, pixels: [2]int{0, 0}},
						{name: "negative clamps to hidden", override: true, width: -3, pixels: [2]int{0, 0}},
					} {
						t.Run(test.name, func(t *testing.T) {
							state := &caretWidthState{area: area, override: test.override, width: test.width}
							tt := newCaretWidthTester(state, scale.value)
							caretWidthWantPixels(t, tt, test.pixels[scale.index])
							active, noIME := tt.TextInputActive()
							if !active || noIME || !tt.Focused("caret target") {
								t.Fatal("paint width changed focus or native text input availability")
							}
						})
					}
				})
			}
		})
	}
}

func TestCaretWidthRemovalAndBlinkUseTheFrameClock(t *testing.T) {
	for _, area := range []bool{false, true} {
		name := "input"
		if area {
			name = "area"
		}
		t.Run(name, func(t *testing.T) {
			state := &caretWidthState{area: area, override: true, width: 2, value: "ab"}
			tt := newCaretWidthTester(state, 2)
			caretWidthWantPixels(t, tt, 4)
			tt.Advance(530 * time.Millisecond)
			caretWidthWantPixels(t, tt, 0)
			tt.Advance(530 * time.Millisecond)
			caretWidthWantPixels(t, tt, 4)
			tt.Advance(530 * time.Millisecond)
			caretWidthWantPixels(t, tt, 0)
			tt.Key(0, ui.KeyLeft)
			caretWidthWantPixels(t, tt, 4)
			state.override = false
			tt.Frame()
			caretWidthWantPixels(t, tt, 2)
			if state.value != "ab" || !tt.Focused("caret target") {
				t.Fatal("removing a paint override changed text or focus")
			}
		})
	}
}

func TestCaretWidthDoesNotChangeEditingIMEOrCandidateGeometry(t *testing.T) {
	for _, area := range []bool{false, true} {
		name := "input"
		if area {
			name = "area"
		}
		t.Run(name, func(t *testing.T) {
			plainState := &caretWidthState{area: area, value: "ab"}
			wideState := &caretWidthState{area: area, value: "ab", override: true, width: 2}
			plain, wide := newCaretWidthTester(plainState, 2), newCaretWidthTester(wideState, 2)
			caretWidthSameEditing(t, plain, wide, plainState, wideState)
			for _, tt := range []*ui.Tester{plain, wide} {
				tt.Key(ui.Shift, ui.KeyLeft)
			}
			caretWidthSameEditing(t, plain, wide, plainState, wideState)
			if plainState.anchor != 2 || plainState.caret != 1 {
				t.Fatalf("fixture did not select the final rune: %d,%d", plainState.anchor, plainState.caret)
			}
			for _, tt := range []*ui.Tester{plain, wide} {
				tt.Advance(530 * time.Millisecond)
			}
			if !bytes.Equal(plain.Image().Pix, wide.Image().Pix) {
				t.Fatal("caret width changed entered text or selection paint while caret blinked off")
			}
			for _, tt := range []*ui.Tester{plain, wide} {
				tt.Type("Z")
			}
			caretWidthSameEditing(t, plain, wide, plainState, wideState)
			if plainState.value != "aZ" {
				t.Fatalf("selection replacement = %q, want aZ", plainState.value)
			}
			for _, tt := range []*ui.Tester{plain, wide} {
				tt.Command("undo")
			}
			caretWidthSameEditing(t, plain, wide, plainState, wideState)
			if plainState.value != "ab" || plainState.anchor != 2 || plainState.caret != 1 {
				t.Fatalf("undo = %q with selection %d,%d, want ab with 2,1", plainState.value, plainState.anchor, plainState.caret)
			}
			for _, tt := range []*ui.Tester{plain, wide} {
				tt.Compose("中文", 1)
			}
			caretWidthSameEditing(t, plain, wide, plainState, wideState)
			// Starting preedit replaces the selected b; the marked text stays
			// separate from the bound value until commit.
			if plainState.value != "a" || !plainState.composing {
				t.Fatalf("IME preedit = value %q, composing %v", plainState.value, plainState.composing)
			}
			for _, tt := range []*ui.Tester{plain, wide} {
				tt.Advance(530 * time.Millisecond)
			}
			if !bytes.Equal(plain.Image().Pix, wide.Image().Pix) {
				t.Fatal("caret width changed preedit glyphs or composition underline")
			}
			for _, tt := range []*ui.Tester{plain, wide} {
				tt.Type("中文")
			}
			caretWidthSameEditing(t, plain, wide, plainState, wideState)
			if plainState.composing || plainState.value != "a中文" {
				t.Fatalf("IME commit = %q, composing %v, want a中文", plainState.value, plainState.composing)
			}
			for _, tt := range []*ui.Tester{plain, wide} {
				tt.Command("undo")
			}
			caretWidthSameEditing(t, plain, wide, plainState, wideState)
			if plainState.value != "ab" || plainState.composing {
				t.Fatalf("undo after IME commit = %q, composing %v", plainState.value, plainState.composing)
			}
		})
	}
}

func TestCaretWidthOverrideDoesNotAffectNeighborOrAnotherWindow(t *testing.T) {
	wideState := &caretWidthState{override: true, width: 3, neighbor: true}
	wide := newCaretWidthTester(wideState, 1)
	caretWidthWantPixels(t, wide, 3)
	if err := wide.Click("caret neighbor"); err != nil {
		t.Fatal(err)
	}
	caretWidthWantPixels(t, wide, 1)
	if err := wide.Click("caret target"); err != nil {
		t.Fatal(err)
	}
	caretWidthWantPixels(t, wide, 3)
	plain := newCaretWidthTester(&caretWidthState{}, 1)
	caretWidthWantPixels(t, plain, 1)
	wide.Frame()
	plain.Frame()
	caretWidthWantPixels(t, wide, 3)
	caretWidthWantPixels(t, plain, 1)
	wideState.width = 0
	wide.Frame()
	plain.Frame()
	caretWidthWantPixels(t, wide, 0)
	caretWidthWantPixels(t, plain, 1)
	wideState.override = false
	wide.Frame()
	caretWidthWantPixels(t, wide, 1)
}

func TestCaretWidthIgnoresNonEditableReusedKey(t *testing.T) {
	state := &caretWidthState{override: true, width: 3, value: "Same text"}
	tt := newCaretWidthTester(state, 1)
	caretWidthWantPixels(t, tt, 3)
	state.textOnly = true
	tt.Frame()
	plainText := newCaretWidthTester(&caretWidthState{textOnly: true, value: state.value}, 1)
	if !bytes.Equal(tt.Image().Pix, plainText.Image().Pix) {
		t.Fatal("CaretWidth changed normal Text reusing an input's editor state")
	}
	state.textOnly, state.override = false, false
	tt.Frame()
	if err := tt.Click("caret target"); err != nil {
		t.Fatal(err)
	}
	caretWidthWantPixels(t, tt, 1)
	if state.value != "Same text" {
		t.Fatal("key reuse changed the bound input value")
	}
}

type caretWidthState struct {
	value          string
	width          float32
	override       bool
	area, textOnly bool
	neighbor       bool
	anchor, caret  int
	composing      bool
}

func newCaretWidthTester(state *caretWidthState, scale float32) *ui.Tester {
	neighbor := ""
	tt := ui.NewTester(func(c *ui.Context) {
		theme := ui.LightTheme()
		theme.Background, theme.Text, theme.TextMuted = ui.RGB(245, 245, 245), ui.RGB(15, 15, 15), ui.RGB(80, 80, 80)
		theme.Accent, theme.Selection = ui.RGB(255, 0, 220), ui.RGB(80, 150, 240)
		c.SetTheme(theme)
		c.Root().FontSize(20)
		ui.Column(c).Fill().Padding(16).Gap(12).AlignItems(ui.Start).Children(func() {
			c.Key("caret target")
			var target ui.Element
			if state.textOnly {
				target = ui.Text(c, state.value)
			} else if state.area {
				target = ui.TextAreaBase(c, &state.value).AutoFocus()
			} else {
				target = ui.TextInputBase(c, &state.value).AutoFocus()
			}
			target.Size(240, 60).Padding(8).Shrink(0).TextColor(theme.Text).Label("caret target")
			if state.override {
				target.CaretWidth(state.width)
			}
			state.anchor, state.caret = target.TextAnchorCaret()
			state.composing = target.Composing()
			if state.neighbor {
				c.Key("caret neighbor")
				ui.TextInputBase(c, &neighbor).Size(240, 30).Padding(4).Shrink(0).TextColor(theme.Text).Label("caret neighbor")
			}
			ui.Text(c, "Ordinary label").TextColor(theme.Text)
			(ui.Element{}).CaretWidth(3)
		})
	}, 320, 180)
	tt.SetScale(scale)
	return tt
}

func caretWidthWantPixels(t *testing.T, tt *ui.Tester, want int) {
	t.Helper()
	img := tt.Image()
	var ink image.Rectangle
	found := false
	for y := range img.Rect.Dy() {
		for x := range img.Rect.Dx() {
			pixel := img.RGBAAt(x, y)
			if pixel.R < 240 || pixel.G > 20 || pixel.B < 200 {
				continue
			}
			cell := image.Rect(x, y, x+1, y+1)
			if found {
				ink = ink.Union(cell)
			} else {
				ink, found = cell, true
			}
		}
	}
	if ink.Dx() != want || want > 0 && ink.Dy() == 0 {
		t.Fatalf("caret raster bounds = %v, want %d device pixels wide", ink, want)
	}
}

func caretWidthSameEditing(t *testing.T, plain, wide *ui.Tester, a, b *caretWidthState) {
	t.Helper()
	if a.value != b.value || a.anchor != b.anchor || a.caret != b.caret || a.composing != b.composing {
		t.Fatalf("paint width changed editor state: plain=%+v wide=%+v", *a, *b)
	}
	first, firstActive := plain.TextCaret()
	second, secondActive := wide.TextCaret()
	if !firstActive || !secondActive || first != second {
		t.Fatalf("paint width changed native candidate geometry: plain=%+v,%v wide=%+v,%v", first, firstActive, second, secondActive)
	}
}
