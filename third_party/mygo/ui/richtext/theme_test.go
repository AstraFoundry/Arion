package richtext

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestThemeResolve checks that what a theme leaves zero takes the
// default, and the colors that follow others follow the theme's.
func TestThemeResolve(t *testing.T) {
	def := DefaultTheme(ui.LightTheme())
	red := ui.RGB(200, 0, 0)
	th := resolve(&Theme{Text: red, CodeSize: 0.8, Headings: [6]HeadingStyle{{Size: 2.5}}}, def)
	if th.Headings[0].Size != 2.5 || th.Headings[0].Weight != 700 || th.Headings[1].Size != def.Headings[1].Size {
		t.Errorf("headings %+v", th.Headings[:2])
	}
	if th.Headings[0].Color != red || th.Marker != red || th.Code != red {
		t.Errorf("heading %v and marker %v don't follow the text %v", th.Headings[0].Color, th.Marker, red)
	}
	if th.CodeSize != 0.8 || th.Indent != def.Indent || th.Link != def.Link || th.QuoteText != def.Muted {
		t.Errorf("resolved %+v", th)
	}
	if got := resolve(nil, def); got != *def {
		t.Errorf("nil theme %+v", got)
	}
}

// TestThemeLook checks that the theme reaches the spans, fonts and room
// of paragraphs.
func TestThemeLook(t *testing.T) {
	link, code, heading := ui.RGB(124, 58, 237), ui.RGB(0, 120, 0), ui.RGB(0, 0, 200)
	th := &Theme{Link: link, CodeBackground: code, Code: code, CodeFont: "Menlo", CodeSize: 0.8, Indent: 2,
		CodeBlockPadding: [2]float32{1, 2}, Headings: [6]HeadingStyle{{Size: 3, Weight: 800, Color: heading, Top: 1, Bottom: 0.5}}}
	v := &view{opts: EditorOptions{Theme: th, Font: ui.Font{Size: 10}}}
	v.setLook(ui.LightTheme(), false)
	look := &v.look
	if s := look.span("x", Attributes{Link: "https://x"}, ParagraphStyle{}); s.Color != link {
		t.Errorf("link %v", s.Color)
	}
	if s := look.span("x", Attributes{Code: true}, ParagraphStyle{}); s.Background != code || s.Font != "Menlo" || s.Size != 8 {
		t.Errorf("code %+v", s)
	}
	if s := look.span("x", Attributes{}, ParagraphStyle{CodeBlock: true}); s.Color != code {
		t.Errorf("code block text %v", s.Color)
	}
	if s := look.span("x", Attributes{}, ParagraphStyle{Heading: 1}); s.Color != heading {
		t.Errorf("heading color %v", s.Color)
	}
	if f := v.paraFont(ParagraphStyle{Heading: 1}); f.Size != 30 || f.Weight != 800 {
		t.Errorf("heading font %+v", f)
	}
	if f := v.paraFont(ParagraphStyle{CodeBlock: true}); f.Family != "Menlo" || f.Size != 8 {
		t.Errorf("code block font %+v", f)
	}
	if left, right := v.insets(ParagraphStyle{CodeBlock: true, Indent: 1}); left != 30 || right != 10 {
		t.Errorf("code block insets %v %v", left, right)
	}
	v.opts.Theme = &Theme{Padding: [2]float32{20, 0}}
	v.setLook(ui.LightTheme(), true)
	if v.padX != 20 || v.padY != ui.LightTheme().Space(2.5) {
		t.Errorf("padding %v %v", v.padX, v.padY)
	}
	v.setLook(ui.LightTheme(), false)
	if v.padX != 0 || v.padY != 0 {
		t.Errorf("an EditorBase's padding %v %v", v.padX, v.padY)
	}
}

// TestThemeChangeLaysOut changes the theme between frames: the text is
// laid out again in it.
func TestThemeChangeLaysOut(t *testing.T) {
	doc := NewDocument(ParseMarkdown("# Title\n\nSome `code` and a [link](https://x)."))
	var th *Theme
	tt := ui.NewTester(func(c *ui.Context) { Editor(c, doc, EditorOptions{Theme: th}).Fill() }, 400, 200)
	shot := func() []byte {
		tt.Frame()
		return bytes.Clone(tt.Image().Pix)
	}
	before := shot()
	th = &Theme{Headings: [6]HeadingStyle{{Size: 2.4}}, Link: ui.RGB(200, 0, 0)}
	if bytes.Equal(shot(), before) {
		t.Error("the theme changed, the text did not")
	}
	th = nil
	if !bytes.Equal(shot(), before) {
		t.Error("back to the default theme, the text shows otherwise")
	}
}
