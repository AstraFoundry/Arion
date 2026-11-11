package ui

import (
	"testing"
	"time"
)

// themedView is a dark window with a light Themed panel, and a text and a
// button after it: the light and dark side by side a theme preview needs.
type themedView struct {
	ctx                       *context
	panel, text, ok, later    *node
	laterOK                   *node
	inside, after             *Theme
	setInside                 *Theme // SetTheme inside the panel, if set
	dark, light               *Theme
	panelSpacing, laterRadius float32
}

func (v *themedView) view(c *context) {
	v.ctx = c
	c.SetTheme(v.dark)
	coreRow(c).Gap(16).AlignItems(Start).Children(func() {
		coreText(c, "dark side")
		v.panel = coreThemed(c, v.light).Padding(8).Children(func() {
			v.inside = c.Theme()
			v.text = coreText(c, "preview")
			v.ok = coreButton(c, "OK")
			if v.setInside != nil {
				c.SetTheme(v.setInside)
			}
		})
		v.after = c.Theme()
		v.later = coreText(c, "later")
		v.laterOK = coreButton(c, "Later")
	})
}

func newThemedView() *themedView {
	return &themedView{dark: DarkTheme(), light: LightTheme()}
}

func TestThemedScopesItsChildren(t *testing.T) {
	v := newThemedView()
	tt := coreNewTester(v.view, 400, 300)
	dark, light := v.dark, v.light
	if got := v.text.resolvedText(); got.color != light.Text || got.size != light.FontSize || got.family != light.Font {
		t.Errorf("bare text in the scope is %v, %v, %q; want the light theme's %v, %v, %q", got.color, got.size, got.family, light.Text, light.FontSize, light.Font)
	}
	if v.ok.bg != light.Surface || v.ok.borderC != light.Border {
		t.Errorf("the button's face is %v bordered %v, want the light theme's %v bordered %v", v.ok.bg, v.ok.borderC, light.Surface, light.Border)
	}
	if label := v.ok.first; label == nil || label.resolvedText().color != light.Text {
		t.Errorf("the button's label is not in the light theme's text color %v", light.Text)
	}
	if v.inside != light {
		t.Errorf("c.Theme() inside the scope is %p, want the scope's %p", v.inside, light)
	}
	// The window keeps its theme.
	root := v.ctx.root
	if root.bg != dark.Background || root.ts.color != dark.Text {
		t.Errorf("the root is %v on %v, want the dark theme's %v on %v", root.ts.color, root.bg, dark.Text, dark.Background)
	}
	if v.after != dark {
		t.Errorf("c.Theme() after the scope is %p, want the window's %p", v.after, dark)
	}
	if got := v.later.resolvedText().color; got != dark.Text {
		t.Errorf("a text after the scope is %v, want the dark theme's %v", got, dark.Text)
	}
	if v.laterOK.bg != dark.Surface {
		t.Errorf("a button after the scope has the face %v, want the dark theme's %v", v.laterOK.bg, dark.Surface)
	}
	// The panel paints the light background, beside the dark window.
	px := func(x, y int) Color {
		c := tt.Image().RGBAAt(x, y)
		return Color{R: c.R, G: c.G, B: c.B, A: c.A}
	}
	p := v.panel
	if got := px(int(p.x+2), int(p.y+2)); got != light.Background {
		t.Errorf("the panel paints %v, want the light background %v", got, light.Background)
	}
	if got := px(int(p.x+p.w+4), 290); got != dark.Background {
		t.Errorf("the window paints %v beside the panel, want the dark background %v", got, dark.Background)
	}
}

func TestThemedTransparent(t *testing.T) {
	dark, light := DarkTheme(), LightTheme()
	var panel *node
	tt := coreNewTester(func(c *context) {
		c.SetTheme(dark)
		panel = coreThemed(c, light).Background(Transparent).Size(100, 100).Children(func() {
			coreText(c, "x")
		})
	}, 200, 200)
	if c := tt.Image().RGBAAt(int(panel.x+90), int(panel.y+90)); (Color{R: c.R, G: c.G, B: c.B, A: c.A}) != dark.Background {
		t.Errorf("a transparent Themed paints %v, want the window's %v behind it", c, dark.Background)
	}
}

func TestThemedNilIsAColumn(t *testing.T) {
	dark := DarkTheme()
	var text *node
	var inside *Theme
	coreNewTester(func(c *context) {
		c.SetTheme(dark)
		coreThemed(c, nil).Children(func() {
			inside = c.Theme()
			text = coreText(c, "x")
		})
	}, 200, 200)
	if inside != dark || text.resolvedText().color != dark.Text {
		t.Errorf("a nil theme scopes %p, text %v; want the window's", inside, text.resolvedText().color)
	}
}

func TestThemedNested(t *testing.T) {
	dark, light := DarkTheme(), LightTheme()
	var outer, inner, back *node
	var themes [3]*Theme
	coreNewTester(func(c *context) {
		coreThemed(c, light).Children(func() {
			outer = coreText(c, "outer")
			themes[0] = c.Theme()
			coreThemed(c, dark).Children(func() {
				themes[1] = c.Theme()
				inner = coreText(c, "inner")
			})
			themes[2] = c.Theme()
			back = coreText(c, "back")
		})
	}, 400, 300)
	if themes != [3]*Theme{light, dark, light} {
		t.Errorf("c.Theme() before, inside and after the inner scope is %p, want %p", themes, [3]*Theme{light, dark, light})
	}
	for _, tc := range []struct {
		name string
		e    *node
		want Color
	}{{"before the inner scope", outer, light.Text}, {"in it", inner, dark.Text}, {"after it", back, light.Text}} {
		if got := tc.e.resolvedText().color; got != tc.want {
			t.Errorf("the text %s is %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestThemedMetrics(t *testing.T) {
	roomy := LightTheme()
	roomy.Spacing, roomy.Radius, roomy.FontSize, roomy.ScrollbarWidth = 6, 12, 20, 11
	var in, out, text *node
	var scroll *node
	var rem float32
	coreNewTester(func(c *context) {
		c.SetTheme(LightTheme())
		out = coreButton(c, "Out")
		coreThemed(c, roomy).Children(func() {
			in = coreButton(c, "In")
			text = coreText(c, "big")
			rem = c.Theme().Rem(1)
			scroll = coreScroll(c).Height(50).Children(func() {
				coreBox(c).Height(500)
			})
		})
	}, 400, 400)
	if in.radius[0] != 12 || out.radius[0] != 6 {
		t.Errorf("the buttons' radii are %v inside and %v outside, want 12 and 6", in.radius[0], out.radius[0])
	}
	// Its padding is the scope's Spacing, and its text the scope's size.
	if in.h <= out.h {
		t.Errorf("the button inside is %v high, not higher than the one outside, %v", in.h, out.h)
	}
	if text.resolvedText().size != 20 || rem != 20 {
		t.Errorf("the text inside is %v large and Rem(1) %v, want the scope's 20", text.resolvedText().size, rem)
	}
	if scroll.st.barWidth != 11 {
		t.Errorf("the scroll bars inside are %v wide, want the scope's 11", scroll.st.barWidth)
	}
}

func TestThemedSetThemeInside(t *testing.T) {
	v := newThemedView()
	other := DarkTheme()
	other.Background = Hex("#101040")
	v.setInside = other
	coreNewTester(v.view, 400, 300)
	if v.ctx.root.bg != other.Background {
		t.Errorf("SetTheme inside a Themed leaves the root %v, want %v", v.ctx.root.bg, other.Background)
	}
	if v.after != other {
		t.Errorf("after the scope, c.Theme() is %p, want the window's new %p", v.after, other)
	}
	if got := v.text.resolvedText().color; got != v.light.Text {
		t.Errorf("the scope's text is %v, want its own theme's %v", got, v.light.Text)
	}
}

func TestThemedPainterTheme(t *testing.T) {
	dark, light := DarkTheme(), LightTheme()
	var in, out *Theme
	coreNewTester(func(c *context) {
		c.SetTheme(dark)
		coreThemed(c, light).Children(func() {
			coreBox(c).Size(10, 10).Draw(func(p *Painter, r Rect) { in = p.Theme() })
		})
		coreBox(c).Size(10, 10).Draw(func(p *Painter, r Rect) { out = p.Theme() })
	}, 200, 200)
	if in != light || out != dark {
		t.Errorf("Painter.Theme() is %p inside and %p outside, want %p and %p", in, out, light, dark)
	}
}

// A popover and a tooltip opened from inside the scope are built in the
// overlay, above the root: they take the scope's theme.
func TestThemedOverlays(t *testing.T) {
	dark, light := DarkTheme(), LightTheme()
	open := true
	var panel, text, outside *node
	var painted *Theme
	tt := coreNewTester(func(c *context) {
		c.SetTheme(dark)
		coreColumn(c).Padding(40).Gap(80).Children(func() {
			coreThemed(c, light).Children(func() {
				anchor := coreButton(c, "Open").Tooltip("Tip")
				panel = corePopover(c, anchor, &open, func() {
					text = coreText(c, "in the popover")
					coreBox(c).Size(4, 4).Draw(func(p *Painter, r Rect) { painted = p.Theme() })
				})
			})
			coreOverlay(c, func() {
				outside = coreText(c, "outside").Absolute().Left(300).Top(250)
			})
		})
	}, 400, 300)
	if panel == nil || panel.bg != light.Background || panel.borderC != light.Border {
		t.Fatalf("the popover's panel is not in the light theme")
	}
	if got := text.resolvedText().color; got != light.Text {
		t.Errorf("bare text in the popover is %v, want the light theme's %v", got, light.Text)
	}
	if painted != light {
		t.Errorf("Painter.Theme() in the popover is %p, want the scope's %p", painted, light)
	}
	if got := outside.resolvedText().color; got != dark.Text {
		t.Errorf("an overlay built outside the scope has text %v, want the window's %v", got, dark.Text)
	}
	// The tooltip is the scope's theme turned over: light's dark text.
	b, _ := tt.Find("Open")
	tt.Move(b.X+b.W/2, b.Y+b.H/2)
	tt.rt.tips.hoverSince = tt.now.Add(-time.Second)
	tt.Frame()
	r, ok := tt.Find("Tip")
	if !ok {
		t.Fatalf("no tooltip; texts %q", tt.Texts())
	}
	px := tt.Image().RGBAAt(int(r.X)-3, int(r.Y+r.H/2))
	if fill, _ := light.inverse(); (Color{R: px.R, G: px.G, B: px.B, A: px.A}) != fill {
		t.Errorf("the tooltip is filled with %v, want the light theme's inverse %v", px, fill)
	}
}

// A copy of a Themed leaving with an exit transition keeps the theme it
// had, though the app changes it.
func TestThemedGhostKeepsTheme(t *testing.T) {
	th := LightTheme()
	g := ghostOf(&node{themed: th, st: &state{}})
	th.Text = Hex("#ff0000")
	if g.themed == th || g.themed.Text == th.Text {
		t.Errorf("the copy shares the app's theme")
	}
}
