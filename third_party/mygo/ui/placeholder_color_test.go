package ui_test

import (
	"bytes"
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestPlaceholderColorDefaultMatchesExplicitThemeTextMuted(t *testing.T) {
	for _, area := range []bool{false, true} {
		name := "single line"
		if area {
			name = "text area"
		}
		t.Run(name, func(t *testing.T) {
			plain := &placeholderColorState{hint: "Placeholder sample", area: area}
			explicit := *plain
			explicit.override, explicit.tint = true, placeholderColorTheme().TextMuted
			before := newPlaceholderColorTester(plain)
			after := newPlaceholderColorTester(&explicit)
			if !bytes.Equal(before.Image().Pix, after.Image().Pix) {
				t.Fatal("default placeholder color differs from explicit Theme.TextMuted")
			}
			plain.hint = ""
			before.Frame()
			if bytes.Equal(before.Image().Pix, after.Image().Pix) {
				t.Fatal("fixture rendered no visible placeholder to compare")
			}
		})
	}
}

func TestPlaceholderColorOverrideChangesOnlyTargetPlaceholder(t *testing.T) {
	state := &placeholderColorState{hint: "Placeholder sample", tint: ui.RGB(240, 40, 80)}
	tt := newPlaceholderColorTester(state)
	before := placeholderColorImage(tt)
	target := placeholderColorBounds(t, tt, "target input")
	neighbor := placeholderColorRegion(t, tt, "neighbor input")
	label := placeholderColorRegion(t, tt, "ordinary label")
	state.override = true
	tt.Frame()
	after := tt.Image()
	changed := 0
	for y := range after.Rect.Dy() {
		for x := range after.Rect.Dx() {
			if before.RGBAAt(x, y) == after.RGBAAt(x, y) {
				continue
			}
			changed++
			if !target.Contains(float32(x)+.5, float32(y)+.5) {
				t.Fatalf("placeholder override changed a pixel outside its input at (%d,%d)", x, y)
			}
		}
	}
	if changed == 0 {
		t.Fatal("explicit placeholder color did not change target glyphs")
	}
	if !bytes.Equal(neighbor, placeholderColorRegion(t, tt, "neighbor input")) || !bytes.Equal(label, placeholderColorRegion(t, tt, "ordinary label")) {
		t.Fatal("placeholder color leaked into another input or normal label")
	}
}

func TestPlaceholderColorTransparentIsAnExplicitOverride(t *testing.T) {
	transparent := newPlaceholderColorTester(&placeholderColorState{hint: "Placeholder sample", override: true, tint: ui.Color{}})
	empty := newPlaceholderColorTester(&placeholderColorState{})
	ordinary := newPlaceholderColorTester(&placeholderColorState{hint: "Placeholder sample"})
	if !bytes.Equal(transparent.Image().Pix, empty.Image().Pix) {
		t.Fatal("transparent Color{} fell back to the theme instead of hiding placeholder ink")
	}
	if bytes.Equal(transparent.Image().Pix, ordinary.Image().Pix) {
		t.Fatal("transparent override did not differ from the visible default")
	}
}

func TestPlaceholderColorRemovalResetsTheSameKeyNextFrame(t *testing.T) {
	state := &placeholderColorState{hint: "Placeholder sample", override: true, tint: ui.RGB(240, 40, 80)}
	tt := newPlaceholderColorTester(state)
	defaultImage := newPlaceholderColorTester(&placeholderColorState{hint: state.hint}).Image()
	if bytes.Equal(tt.Image().Pix, defaultImage.Pix) {
		t.Fatal("fixture needs a visible override before resetting")
	}
	state.override = false
	tt.Frame()
	if !bytes.Equal(tt.Image().Pix, defaultImage.Pix) {
		t.Fatal("omitting PlaceholderColor retained a prior frame's override")
	}
	tt.Frame()
	if !bytes.Equal(tt.Image().Pix, defaultImage.Pix) {
		t.Fatal("default placeholder color changed on a later frame")
	}
}

func TestPlaceholderColorDoesNotChangeEnteredTextOrIMEComposition(t *testing.T) {
	t.Run("entered text", func(t *testing.T) {
		plain := newPlaceholderColorTester(&placeholderColorState{hint: "Placeholder sample", value: "Entered text"})
		colored := newPlaceholderColorTester(&placeholderColorState{hint: "Placeholder sample", value: "Entered text", override: true, tint: ui.RGB(240, 40, 80)})
		if !bytes.Equal(plain.Image().Pix, colored.Image().Pix) {
			t.Fatal("placeholder override changed entered text paint")
		}
	})
	t.Run("IME composition", func(t *testing.T) {
		plainState := &placeholderColorState{hint: "Placeholder sample"}
		colorState := &placeholderColorState{hint: "Placeholder sample", override: true, tint: ui.RGB(240, 40, 80)}
		plain, colored := newPlaceholderColorTester(plainState), newPlaceholderColorTester(colorState)
		for _, tt := range []*ui.Tester{plain, colored} {
			if err := tt.Click("target input"); err != nil {
				t.Fatal(err)
			}
			tt.Compose("preedit", 3)
		}
		if plainState.value != "" || colorState.value != "" {
			t.Fatal("fixture committed composition instead of testing preedit")
		}
		if !bytes.Equal(plain.Image().Pix, colored.Image().Pix) {
			t.Fatal("placeholder override changed uncommitted IME text paint")
		}
	})
}

func TestPlaceholderColorIgnoresNonEditableReusedKeys(t *testing.T) {
	state := &placeholderColorState{hint: "Placeholder sample", override: true, tint: ui.RGB(240, 40, 80)}
	tt := newPlaceholderColorTester(state)
	state.textOnly = true
	tt.Frame()
	plainText := newPlaceholderColorTester(&placeholderColorState{textOnly: true})
	if !bytes.Equal(tt.Image().Pix, plainText.Image().Pix) {
		t.Fatal("PlaceholderColor affected normal Text reusing an input's editor state")
	}
	state.textOnly, state.override = false, false
	tt.Frame()
	plainInput := newPlaceholderColorTester(&placeholderColorState{hint: state.hint})
	if !bytes.Equal(tt.Image().Pix, plainInput.Image().Pix) {
		t.Fatal("reused input inherited a non-input placeholder override")
	}
}

func TestPlaceholderColorStaysWithinItsWindow(t *testing.T) {
	plainState := &placeholderColorState{hint: "Placeholder sample"}
	plain := newPlaceholderColorTester(plainState)
	before := bytes.Clone(plain.Image().Pix)
	colorState := &placeholderColorState{hint: "Placeholder sample", override: true, tint: ui.RGB(240, 40, 80)}
	colored := newPlaceholderColorTester(colorState)
	if bytes.Equal(plain.Image().Pix, colored.Image().Pix) {
		t.Fatal("fixture needs different placeholder colors across windows")
	}
	plain.Frame()
	if !bytes.Equal(before, plain.Image().Pix) {
		t.Fatal("another window's placeholder override changed this window")
	}
	colored.Frame()
	plain.Frame()
	if !bytes.Equal(before, plain.Image().Pix) {
		t.Fatal("rebuilding another window leaked its override")
	}
	colorState.override = false
	colored.Frame()
	if !bytes.Equal(before, colored.Image().Pix) {
		t.Fatal("clearing the other window's override failed to restore its own default")
	}
}

type placeholderColorState struct {
	hint, value string
	override    bool
	area        bool
	textOnly    bool
	tint        ui.Color
}

func newPlaceholderColorTester(state *placeholderColorState) *ui.Tester {
	neighbor := ""
	return ui.NewTester(func(c *ui.Context) {
		theme := placeholderColorTheme()
		c.SetTheme(theme)
		c.Root().FontSize(18)
		ui.Column(c).Fill().Padding(8).Gap(8).AlignItems(ui.Start).Children(func() {
			c.Key("target")
			var target ui.Element
			if state.textOnly {
				target = ui.Text(c, "Plain target text")
			} else if state.area {
				target = ui.TextAreaBase(c, &state.value).Placeholder(state.hint)
			} else {
				target = ui.TextInputBase(c, &state.value).Placeholder(state.hint)
			}
			target.Size(200, 30).Shrink(0).TextColor(theme.Text).Label("target input")
			if state.override {
				target.PlaceholderColor(state.tint)
			}
			c.Key("neighbor")
			ui.TextInputBase(c, &neighbor).Size(200, 30).Shrink(0).TextColor(theme.Text).Placeholder("Neighbor placeholder").Label("neighbor input")
			ui.Text(c, "Ordinary label").Size(200, 30).Shrink(0).TextColor(theme.Text).Label("ordinary label")
			(ui.Element{}).PlaceholderColor(ui.RGB(240, 40, 80))
		})
	}, 240, 140)
}

func placeholderColorTheme() *ui.Theme {
	theme := ui.LightTheme()
	theme.Background = ui.RGB(20, 25, 35)
	theme.Text, theme.TextMuted = ui.RGB(225, 230, 235), ui.RGB(95, 145, 205)
	return theme
}

func placeholderColorBounds(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	bounds, ok := tt.Find(label)
	if !ok {
		t.Fatalf("missing %q", label)
	}
	return bounds
}

func placeholderColorRegion(t *testing.T, tt *ui.Tester, label string) []byte {
	t.Helper()
	bounds := placeholderColorBounds(t, tt, label)
	img := tt.Image()
	var pixels []byte
	for y := int(bounds.Y); y < int(bounds.Y+bounds.H); y++ {
		start, end := img.PixOffset(int(bounds.X), y), img.PixOffset(int(bounds.X+bounds.W), y)
		pixels = append(pixels, img.Pix[start:end]...)
	}
	return pixels
}

func placeholderColorImage(tt *ui.Tester) *image.RGBA {
	source := tt.Image()
	copy := *source
	copy.Pix = bytes.Clone(source.Pix)
	return &copy
}
