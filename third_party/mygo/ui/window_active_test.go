package ui

import (
	"image"
	"math"
	"testing"
)

// TestWindowActive checks that a view reads whether its window is drawn as
// active, and is built anew as it changes only when its last frame read
// it, as a window that loses the keyboard draws no frame that shows what
// showed already.
func TestWindowActive(t *testing.T) {
	read, builds := true, 0
	var active bool
	tt := NewTester(func(c *Context) {
		builds++
		if read {
			active = c.WindowActive()
		}
		Text(c, "content")
	}, 200, 100)
	if !active {
		t.Fatal("a window starts inactive")
	}
	tt.SetWindowActive(false)
	if active {
		t.Error("WindowActive is true in a window drawn inactive")
	}
	tt.SetWindowActive(true)
	if !active {
		t.Error("WindowActive is false in a window drawn active again")
	}

	// The keyboard leaving the window, as for one of its flyouts, leaves
	// it active.
	tt.SetFocused(false)
	tt.Frame()
	if !active {
		t.Error("WindowActive follows the keyboard focus")
	}
	tt.SetFocused(true)

	read = false
	tt.Frame()
	before := builds
	tt.h.inactive = true
	tt.rt.activeChanged()
	if tt.FramePending() {
		t.Error("a change asks for a frame of a view that did not read it")
	}
	read = true
	tt.Frame()
	tt.h.inactive = false
	tt.rt.activeChanged()
	if !tt.FramePending() {
		t.Error("a change asks for no frame of a view that read it")
	}
	tt.Frame()
	if !active || builds != before+2 {
		t.Errorf("active %v after %d builds, want true after 2", active, builds-before)
	}
}

// TestWindowActiveFadesAccent checks that the built-in widgets fade their
// accent to a neutral gray in a window that is not drawn as active, as
// AppKit's do, and keep it with a theme whose InactiveAccent is Accent.
func TestWindowActiveFadesAccent(t *testing.T) {
	icon := MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><rect width="16" height="16" fill="currentColor"/></svg>`))
	checked, on, value, place := true, true, 1.0, "mail"
	var theme *Theme
	tt := coreNewTester(func(c *context) {
		if theme != nil {
			c.SetTheme(theme)
		}
		coreRow(c).Fill().AlignItems(Stretch).Children(func() {
			coreSidebar(c, &place, func() {
				coreSidebarSection(c, "Places", nil, func() {
					coreSidebarItem(c, "mail", nil, "Mail")
					coreSidebarItem(c, "notes", icon, "Notes").Label("notes")
				})
			}).Width(120)
			coreColumn(c).Padding(10).Gap(10).Children(func() {
				corePrimaryButton(c, "Send").Width(100).Label("send")
				coreCheckbox(c, &checked, "Agree").Label("agree")
				coreSwitch(c, &on).Label("wifi")
				coreSlider(c, &value, 0, 1).Width(100).Label("volume")
				coreProgress(c, 1).Width(100).Label("copying")
				coreBox(c).Size(100, 20).Selected(true).Label("row")
			})
		})
	}, 400, 300)
	tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})

	// Where each widget fills with the accent, in the window's DIPs.
	spots := map[string]func(r Rect) (float32, float32){
		"send":    func(r Rect) (float32, float32) { return r.X + 3, r.Y + r.H/2 },
		"agree":   func(r Rect) (float32, float32) { return r.X + 3, r.Y + 4 }, // its box, off the check
		"wifi":    func(r Rect) (float32, float32) { return r.X + 4, r.Y + r.H/2 },
		"volume":  func(r Rect) (float32, float32) { return r.X + r.W/2, r.Y + 10 },
		"copying": func(r Rect) (float32, float32) { return r.X + r.W/2, r.Y + r.H/2 },
		"row":     func(r Rect) (float32, float32) { return r.X + r.W/2, r.Y + r.H/2 },
		"notes":   func(r Rect) (float32, float32) { return r.X + 16, r.Y + r.H/2 }, // its icon
	}
	colors := func() map[string]Color {
		img := tt.Image()
		got := map[string]Color{}
		for name, spot := range spots {
			r, ok := tt.Find(name)
			if !ok {
				t.Fatalf("no %s: %q", name, tt.Texts())
			}
			x, y := spot(r)
			got[name] = pixel(img, x, y)
		}
		return got
	}
	light := LightTheme()
	gray := light.inactiveAccent()
	for name, c := range colors() {
		if c != light.Accent {
			t.Errorf("active: %s shows %v, want the accent %v", name, c, light.Accent)
		}
	}

	tt.SetWindowActive(false)
	for name, c := range colors() {
		want := gray
		if name == "send" {
			want = light.Surface // as other buttons
		}
		if c != want {
			t.Errorf("inactive: %s shows %v, want %v", name, c, want)
		}
	}

	// A theme keeping the accent in every window.
	kept := LightTheme()
	kept.InactiveAccent = kept.Accent
	theme = kept
	tt.Frame()
	for name, c := range colors() {
		if c != kept.Accent {
			t.Errorf("inactive, accent kept: %s shows %v, want the accent %v", name, c, kept.Accent)
		}
	}
	theme = nil

	// The gray stands out under the accent's text, in both appearances.
	for _, th := range []*Theme{LightTheme(), DarkTheme()} {
		if g := th.inactiveAccent(); contrast(g, th.AccentText) < 4.5 {
			t.Errorf("dark %v: %v under %v has a contrast of %.2f", th.Dark, g, th.AccentText, contrast(g, th.AccentText))
		}
	}
}

// pixel returns the color of the pixel at x, y, in DIPs of a frame drawn at
// scale 1.
func pixel(img *image.RGBA, x, y float32) Color {
	c := img.RGBAAt(int(x), int(y))
	return Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

// contrast returns WCAG's contrast ratio of two opaque colors.
func contrast(a, b Color) float64 {
	l := func(c Color) float64 {
		lin := func(v uint8) float64 {
			s := float64(v) / 255
			if s <= 0.04045 {
				return s / 12.92
			}
			return math.Pow((s+0.055)/1.055, 2.4)
		}
		return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
	}
	x, y := l(a)+0.05, l(b)+0.05
	if x < y {
		x, y = y, x
	}
	return x / y
}

// TestWindowActiveKeepsLayout checks that a primary button, drawn as other
// buttons in a window not drawn as active, keeps its size, so that what is
// around it does not move as the window loses or takes the keyboard.
func TestWindowActiveKeepsLayout(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(10).Gap(10).Children(func() {
			corePrimaryButton(c, "Send").Label("send")
			coreText(c, "below").Label("below")
		})
	}, 300, 200)
	boxes := func() (Rect, Rect) {
		b, ok := tt.Find("send")
		if !ok {
			t.Fatalf("no button: %q", tt.Texts())
		}
		l, ok := tt.Find("below")
		if !ok {
			t.Fatalf("no text: %q", tt.Texts())
		}
		return b, l
	}
	b0, l0 := boxes()
	tt.SetWindowActive(false)
	b1, l1 := boxes()
	if b1 != b0 || l1 != l0 {
		t.Errorf("inactive, the button is %v and the text below %v; active, %v and %v", b1, l1, b0, l0)
	}
}
