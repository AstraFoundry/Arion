//go:build !mygo_noinspector

package ui

import (
	"image/color"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// styled returns an inspector showing a box keyed "box", chosen, and the
// box's text inside a column that sets its text color, for the view.
func styled(t *testing.T) *Tester {
	t.Helper()
	tt, _ := inspected(func(c *context) {
		coreColumn(c).Padding(20).TextColor(RGB(0, 128, 0)).Children(func() {
			coreColumn(c).Key("box").Width(100).Padding(4).Background(RGB(255, 0, 0)).Children(func() {
				coreText(c, "Inside")
			})
		})
	})
	openInspector(tt)
	return tt
}

// chooseKeyed chooses the element keyed k.
func chooseKeyed(t *testing.T, tt *Tester, k string) *node {
	t.Helper()
	in := &tt.rt.insp
	for id, key := range in.keys {
		if key == k {
			in.choose(id)
			tt.Frame()
			return elementOf(t, tt, id)
		}
	}
	t.Fatalf("no element is keyed %q: %v", k, in.keys)
	return nil
}

// elementOf returns the element id of the last frame.
func elementOf(t *testing.T, tt *Tester, id uint64) *node {
	t.Helper()
	var found *node
	var walk func(e *node)
	walk = func(e *node) {
		if e.id == id {
			found = e
		}
		for c := e.first; c != nil; c = c.next {
			walk(c)
		}
	}
	walk(tt.rt.c.root)
	if found == nil {
		t.Fatalf("no element has the ID %x", id)
	}
	return found
}

// clickValue clicks the value of a property in the Styles tab.
func clickValue(t *testing.T, tt *Tester, name string) {
	t.Helper()
	for _, s := range tt.Texts() {
		if strings.HasPrefix(s, "Value of "+name+": ") {
			if err := tt.Click(s); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("no value of %s: %q", name, tt.Texts())
}

func pixelAt(tt *Tester, e *node) color.RGBA {
	return tt.Image().RGBAAt(int(e.x)+2, int(e.y)+2)
}

func TestInspectorEditsAValue(t *testing.T) {
	tt := styled(t)
	e := chooseKeyed(t, tt, "box")
	clickValue(t, tt, "background-color")
	if !tt.rt.insp.editing.active {
		t.Fatal("clicking the value does not edit it")
	}
	// What is typed replaces the value, and shows as it is typed.
	tt.Type("#0000ff")
	if c := pixelAt(tt, elementOf(t, tt, e.id)); c.B != 255 || c.R != 0 {
		t.Errorf("the window does not preview the color: %v", c)
	}
	tt.Key(0, KeyEnter)
	if tt.rt.insp.editing.active {
		t.Error("Enter did not end editing")
	}
	tt.Frame()
	if !tt.HasText("#0000ff") || !tt.HasText("1 change") {
		t.Errorf("the tab does not show the change: %q", tt.Texts())
	}
	if c := pixelAt(tt, elementOf(t, tt, e.id)); c.B != 255 {
		t.Errorf("the change did not stay: %v", c)
	}
	// Reset gives back what the view builds.
	if err := tt.Click("Reset"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if c := pixelAt(tt, elementOf(t, tt, e.id)); c.R != 255 || c.B != 0 {
		t.Errorf("Reset did not bring the color back: %v", c)
	}
	if tt.HasText("1 change") {
		t.Errorf("the changes outlast Reset: %q", tt.Texts())
	}
}

func TestInspectorEscapeGoesBack(t *testing.T) {
	tt := styled(t)
	e := chooseKeyed(t, tt, "box")
	clickValue(t, tt, "width")
	tt.Type("300")
	if got := elementOf(t, tt, e.id).w; got != 300 {
		t.Errorf("width previews as %v", got)
	}
	tt.Key(0, KeyEscape)
	tt.Frame()
	if got := elementOf(t, tt, e.id).w; got != 100 {
		t.Errorf("after Escape, the width is %v", got)
	}
	if n := tt.rt.insp.changes(); n != 0 {
		t.Errorf("Escape left %d changes", n)
	}
}

func TestInspectorTurnsDeclarationsOff(t *testing.T) {
	tt := styled(t)
	e := chooseKeyed(t, tt, "box")
	if err := tt.Click("width on"); err != nil {
		t.Fatalf("%v: %q", err, tt.Texts())
	}
	tt.Frame()
	if got := elementOf(t, tt, e.id); got.width.u != unitAuto {
		t.Errorf("turned off, width is still %v", got.width)
	}
	// It stays in the tab, in its place, to turn on again.
	rule := tt.rt.insp.sel.rules[0]
	if rule.decls[0].name != "width" || !rule.decls[0].off {
		t.Errorf("the rule lists %+v", rule.decls)
	}
	if err := tt.Click("width on"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := elementOf(t, tt, e.id).w; got != 100 {
		t.Errorf("turned on, the width is %v", got)
	}
	if n := tt.rt.insp.changes(); n != 0 {
		t.Errorf("turned on again, it left %d changes", n)
	}
}

func TestInspectorInheritedDeclarations(t *testing.T) {
	tt := styled(t)
	chooseKeyed(t, tt, "box")
	if !tt.HasText("Inherited from") {
		t.Fatalf("no inherited rule: %q", tt.Texts())
	}
	parent := tt.rt.insp.sel.rules[1]
	// Turning it off changes the element that sets it, and stays listed
	// though that element sets nothing any more.
	if err := tt.Click("color on"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := elementOf(t, tt, parent.id); got.ts.set&setColor != 0 {
		t.Error("turning off the inherited color did not change the element setting it")
	}
	if !tt.HasText("Inherited from") {
		t.Errorf("the rule went with its only declaration: %q", tt.Texts())
	}
	if err := tt.Click("color on"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := elementOf(t, tt, parent.id); got.ts.set&setColor == 0 {
		t.Error("the inherited color did not come back")
	}
}

func TestInspectorAddsDeclarations(t *testing.T) {
	tt := styled(t)
	e := chooseKeyed(t, tt, "box")
	if err := tt.Click("Add a style"); err != nil {
		t.Fatal(err)
	}
	tt.Type("opacity: 0.5; nonsense: 1")
	tt.Key(0, KeyEnter)
	tt.Frame()
	if got := elementOf(t, tt, e.id); !got.opacitySet || got.opacity != 0.5 {
		t.Errorf("opacity is %v", got.opacity)
	}
	// What it does not understand shows, marked.
	var invalid bool
	for _, d := range tt.rt.insp.sel.rules[0].decls {
		invalid = invalid || d.name == "nonsense" && d.invalid
	}
	if !invalid {
		t.Errorf("the unknown property is not marked: %+v", tt.rt.insp.sel.rules[0].decls)
	}
}

func TestInspectorArrowsMoveANumber(t *testing.T) {
	tt := styled(t)
	e := chooseKeyed(t, tt, "box")
	clickValue(t, tt, "width")
	tt.Key(0, KeyUp)
	if got := elementOf(t, tt, e.id).w; got != 101 {
		t.Errorf("Up: the width is %v", got)
	}
	tt.Key(Shift, KeyDown)
	if got := elementOf(t, tt, e.id).w; got != 91 {
		t.Errorf("Shift+Down: the width is %v", got)
	}
}

func TestInspectorCloseDropsEdits(t *testing.T) {
	tt := styled(t)
	e := chooseKeyed(t, tt, "box")
	id := e.id
	clickValue(t, tt, "width")
	tt.Type("300")
	tt.Key(0, KeyEnter)
	tt.rt.toggleInspector()
	tt.Frame()
	if got := elementOf(t, tt, id).w; got != 100 {
		t.Errorf("after closing, the width is %v", got)
	}
}

func TestSetStyleRoundTripsOwnStyles(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"flex-direction", "row-reverse"},
		{"flex-wrap", "wrap"},
		{"gap", "8px"},
		{"justify-content", "space-between"},
		{"align-items", "center"},
		{"width", "50%"},
		{"max-height", "20px"},
		{"flex-grow", "2"},
		{"padding", "4px 8px"},
		{"margin", "auto"},
		{"border", "1px dashed #112233"},
		{"border-radius", "6px"},
		{"filter", "blur(3px)"},
		{"background-color", "#abcdef"},
		{"box-shadow", "inset 1px 2px 3px 4px #00000080"},
		{"opacity", "0.5"},
		{"color", "#102030"},
		{"font-size", "13px"},
		{"font-weight", "600"},
		{"line-height", "18px"},
		{"text-decoration", "underline line-through"},
		{"z-index", "-2"},
	} {
		var decls []inspDecl
		understood := false
		tt := coreNewTester(func(c *context) {
			e := coreColumn(c)
			understood = setStyle(e, tc.name, tc.value)
			decls = ownStyles(c.rt, e)
		}, 100, 100)
		tt.Frame()
		if !understood {
			t.Errorf("setStyle does not understand %s: %s", tc.name, tc.value)
			continue
		}
		found := false
		for _, d := range decls {
			found = found || d.name == tc.name && d.value == tc.value
		}
		if !found {
			t.Errorf("%s: %s reads back as %+v", tc.name, tc.value, decls)
		}
	}
	for _, tc := range []struct{ name, value string }{
		{"width", "wide"}, {"padding", "1 2 3 4 5"}, {"opacity", "much"}, {"color", "#12"},
		{"nonsense", "1"}, {"box-shadow", "1px"}, {"font-size", "-1"}, {"z-index", "1.5"},
	} {
		if setStyle(&node{}, tc.name, tc.value) {
			t.Errorf("setStyle understood %s: %s", tc.name, tc.value)
		}
	}
}

func TestStyleKnownCoversOwnStyles(t *testing.T) {
	// Every known property turns off without setStyle's help, and setStyle
	// knows no others.
	e := &node{kind: kindBox, shrink: 1}
	for _, name := range []string{"color", "padding", "box-shadow", "white-space"} {
		if !styleKnown(name) {
			t.Errorf("%s is not known", name)
		}
		unsetStyle(e, name)
	}
	if styleKnown("attach") {
		t.Error("attach is known, which cannot be set")
	}
}

func TestParseCSS(t *testing.T) {
	for s, want := range map[string]Color{
		"#fff":             RGB(255, 255, 255),
		"rgb(1, 2, 3)":     RGB(1, 2, 3),
		"rgb(1 2 3 / 0.5)": RGBA(1, 2, 3, 0.5),
		"Red":              RGB(255, 0, 0),
		"transparent":      Transparent,
	} {
		if c, ok := parseCSSColor(s); !ok || c != want {
			t.Errorf("parseCSSColor(%q) = %v, %v", s, c, ok)
		}
	}
	if _, ok := parseCSSColor("nope"); ok {
		t.Error("parseCSSColor took nope")
	}
	if v, ok := parseCSSEdges("1 auto", true); !ok || v[0] != 1 || !math.IsInf(float64(v[1]), 1) || v[2] != 1 {
		t.Errorf("parseCSSEdges = %v, %v", v, ok)
	}
	for _, tc := range []struct {
		text string
		by   float64
		want string
	}{
		{"8px", 1, "9px"},
		{"8px 12px", -1, "7px 12px"},
		{"1px solid #111", 10, "11px solid #111"},
		{"#112233", 1, "#112233"},
		{"0.5", 0.1, "0.6"},
		{"-2px", 1, "-1px"},
		{"auto", 1, "auto"},
	} {
		if got := nudge(tc.text, tc.by); got != tc.want {
			t.Errorf("nudge(%q, %v) = %q, not %q", tc.text, tc.by, got, tc.want)
		}
	}
}

func TestInspectorCopiesChangesAsGo(t *testing.T) {
	tt := styled(t)
	chooseKeyed(t, tt, "box")
	clickValue(t, tt, "background-color")
	tt.Type("#fde68a")
	tt.Key(0, KeyEnter)
	if err := tt.Click("width on"); err != nil {
		t.Fatal(err)
	}
	clickValue(t, tt, "padding")
	tt.Type("8px 12px")
	tt.Key(0, KeyEnter)
	if err := tt.Click("Add a style"); err != nil {
		t.Fatal(err)
	}
	tt.Type("opacity: lots")
	tt.Key(0, KeyEnter)
	tt.Frame()
	// Package ui's own frames have no source, unlike an app's.
	tt.rt.insp.sources = map[uint64]string{tt.rt.insp.selected: "/app/main.go:42"}
	if err := tt.Click("Copy as Go"); err != nil {
		t.Fatalf("%v: %q", err, tt.Texts())
	}
	got := tt.Clipboard()
	for _, want := range []string{
		"// Column#box, main.go:42\n",
		"// Remove width: 100px\n",
		"// Not understood, opacity: lots\n",
		`.Background(ui.Hex("#fde68a")).Padding(8, 12)`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the code lacks %q:\n%s", want, got)
		}
	}
}

func TestGoCallsAreElementMethods(t *testing.T) {
	methods := reflect.TypeFor[Element]()
	callRE := regexp.MustCompile(`\.([A-Z]\w*)\(`)
	for _, tc := range []struct{ name, value, want string }{
		{"flex-direction", "row-reverse", ".Row().Reverse()"},
		{"flex-direction", "column", ".Column()"},
		{"flex-wrap", "wrap-reverse", ".WrapReverse()"},
		{"gap", "8", ".Gap(8)"},
		{"gap", "8 4", ".GapY(8).GapX(4)"},
		{"row-gap", "8", ".GapY(8)"},
		{"column-gap", "8", ".GapX(8)"},
		{"justify-content", "space-between", ".Justify(ui.SpaceBetween)"},
		{"align-items", "center", ".AlignItems(ui.Center)"},
		{"align-content", "stretch", ".AlignContent(ui.Stretch)"},
		{"align-self", "end", ".AlignSelf(ui.End)"},
		{"width", "12px", ".Width(12)"},
		{"width", "50%", ".WidthPercent(50)"},
		{"height", "50%", ".HeightPercent(50)"},
		{"min-width", "1", ".MinWidth(1)"},
		{"min-height", "1%", ".MinHeightPercent(1)"},
		{"max-width", "1", ".MaxWidth(1)"},
		{"max-height", "1%", ".MaxHeightPercent(1)"},
		{"max-width", "min(78%, 544px)", ".MaxWidthPercent(78).MaxWidth(544)"},
		{"min-width", "max(20px, 10%)", ".MinWidthPercent(10).MinWidth(20)"},
		{"flex-basis", "1%", ".BasisPercent(1)"},
		{"top", "1", ".Top(1)"},
		{"right", "1", ".Right(1)"},
		{"bottom", "1%", ".BottomPercent(1)"},
		{"left", "1%", ".LeftPercent(1)"},
		{"aspect-ratio", "16 / 8", ".AspectRatio(2)"},
		{"flex-grow", "2", ".Grow(2)"},
		{"flex-shrink", "0", ".Shrink(0)"},
		{"padding", "1 2 3 4", ".Padding(1, 2, 3, 4)"},
		{"margin", "auto", ".Margin(ui.Auto)"},
		{"border", "1px solid #111111", `.Border(1, ui.Hex("#111111"))`},
		{"border", "0 0 1px 0 dashed red", `.BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#ff0000")).BorderStyle(ui.BorderDashed)`},
		{"border-width", "2", ".BorderWidth(2)"},
		{"border-color", "white", `.BorderColor(ui.Hex("#ffffff"))`},
		{"border-style", "dashed", ".BorderStyle(ui.BorderDashed)"},
		{"border-radius", "6", ".Radius(6)"},
		{"border-radius", "1 2", ".Radius(1, 2, 1, 2)"},
		{"filter", "blur(3px)", ".Blur(3)"},
		{"background", "transparent", ".Background(ui.Transparent)"},
		{"box-shadow", "1 2 3 4 #000000", `.Shadow(1, 2, 3, 4, ui.Hex("#000000"))`},
		{"box-shadow", "inset 1 2 #000000", `.InsetShadow(1, 2, 0, 0, ui.Hex("#000000"))`},
		{"box-shadow", "none", ".NoShadow()"},
		{"opacity", "50%", ".Opacity(0.5)"},
		{"visibility", "hidden", ".Invisible()"},
		{"display", "none", ".Hidden(true)"},
		{"color", "oklch(0.7 0.3 150 / 0.5)", ""},
		{"font-family", "'Menlo'", `.Font("Menlo")`},
		{"font-size", "13", ".FontSize(13)"},
		{"font-weight", "bold", ".FontWeight(700)"},
		{"font-style", "italic", ".Italic()"},
		{"line-height", "18px", ".FixedLineHeight(18)"},
		{"line-height", "1.5", ".LineHeight(1.5)"},
		{"letter-spacing", "1", ".LetterSpacing(1)"},
		{"text-align", "center", ".TextAlign(ui.Center)"},
		{"text-decoration", "underline wavy", ".WavyUnderline()"},
		{"text-decoration", "underline dotted line-through", ".DottedUnderline().Strikethrough()"},
		{"font-feature-settings", "tnum", `.FontFeatures("tnum")`},
		{"selection-color", "#000", `.SelectionColor(ui.Hex("#000000"))`},
		{"caret-color", "#000", `.CaretColor(ui.Hex("#000000"))`},
		{"line-clamp", "2", ".MaxLines(2)"},
		{"white-space", "nowrap", ".NoWrap()"},
		{"z-index", "3", ".ZIndex(3)"},
		{"z-index", "auto", ".ZIndex(0)"},
	} {
		code := strings.Join(goCalls(tc.name, tc.value), "")
		if tc.want != "" && code != tc.want {
			t.Errorf("%s: %s is %q, not %q", tc.name, tc.value, code, tc.want)
		}
		if code == "" {
			t.Errorf("%s: %s has no call", tc.name, tc.value)
		}
		for _, m := range callRE.FindAllStringSubmatch(code, -1) {
			if _, ok := methods.MethodByName(m[1]); !ok && m[1] != "Oklch" && m[1] != "Hex" && m[1] != "Alpha" {
				t.Errorf("%s: %s calls %s, which Element lacks", tc.name, tc.value, m[1])
			}
		}
	}
	// A wide color stays wide, with its alpha.
	c := Oklch(0.7, 0.3, 150).Alpha(0.5)
	if got := goColor(c); !strings.HasPrefix(got, "ui.Oklch(0.") || !strings.HasSuffix(got, ").Alpha(0.5)") {
		t.Errorf("a wide color is %q", got)
	}
}

func TestInspectorForcesStates(t *testing.T) {
	var hovered, pressed, focused, visible, within bool
	tt, _ := inspected(func(c *context) {
		coreColumn(c).Padding(20).Children(func() {
			b := coreBox(c).Key("box").Size(40, 40).Focusable()
			hovered, pressed = b.Hovered(), b.Pressed()
			focused, visible, within = b.Focused(), b.FocusVisible(), b.FocusWithin()
		})
	})
	openInspector(tt)
	e := chooseKeyed(t, tt, "box")
	if hovered || pressed || focused || visible || within {
		t.Fatal("the box starts in a state")
	}
	if tt.HasText(":hover") {
		t.Errorf("the bar shows before it opens: %q", tt.Texts())
	}
	if err := tt.Click(":hov"); err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct {
		name string
		on   *bool
	}{{":hover", &hovered}, {":active", &pressed}, {":focus", &focused}, {":focus-visible", &visible}, {":focus-within", &within}} {
		if err := tt.Click(s.name); err != nil {
			t.Fatalf("%v: %q", err, tt.Texts())
		}
		if !*s.on {
			t.Errorf("%s does not reach the view", s.name)
		}
		if !tt.HasText(s.name) {
			t.Errorf("%s does not show: %q", s.name, tt.Texts())
		}
	}
	if tt.rt.focused == e.id {
		t.Error("forcing the focus moved the real focus")
	}
	// Forcing goes, one at a time, and with the inspector.
	if err := tt.Click(":hover"); err != nil {
		t.Fatal(err)
	}
	if hovered {
		t.Error(":hover stayed forced")
	}
	tt.rt.toggleInspector()
	tt.Frame()
	if pressed || focused || visible || within {
		t.Error("the states outlast the inspector")
	}
}

func TestInspectorForcedFocusRings(t *testing.T) {
	tt, _ := inspected(func(c *context) {
		coreColumn(c).Padding(20).Children(func() { coreBox(c).Key("box").Size(40, 40).Focusable() })
	})
	openInspector(tt)
	e := chooseKeyed(t, tt, "box")
	if elementOf(t, tt, e.id).ringShown() {
		t.Fatal("the box has a ring")
	}
	tt.rt.insp.toggleForced(e.id, forceFocusVisible)
	tt.Frame()
	if !elementOf(t, tt, e.id).ringShown() {
		t.Error(":focus-visible does not ring the box")
	}
}

// The inspector writes a bound in DIPs and by percentage as CSS's min()
// or max() of the two, and reads it back.
func TestCSSBoundRoundTrip(t *testing.T) {
	var both, pct bound
	both.setPct(78)
	both.setPx(544)
	pct.setPct(50)
	for _, tc := range []struct {
		b     bound
		upper bool
		css   string
	}{{both, true, "min(78%, 544px)"}, {both, false, "max(78%, 544px)"}, {pct, true, "50%"}} {
		css := cssBound(tc.b, tc.upper)
		if css != tc.css {
			t.Errorf("%+v is %q, want %q", tc.b, css, tc.css)
		}
		if b, ok := parseCSSBound(css, tc.upper); !ok || b != tc.b {
			t.Errorf("%q reads back as %+v, %v", css, b, ok)
		}
	}
	for _, s := range []string{"min(1px, 2px)", "max(78%, 544px)", "min(auto, 5px)", "min(5px)"} {
		if _, ok := parseCSSBound(s, true); ok {
			t.Errorf("max-width: %s was understood", s)
		}
	}
}
