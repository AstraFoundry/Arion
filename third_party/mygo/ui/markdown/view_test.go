package markdown

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const sample = "# Title\n\nSome **bold** and a [link](https://go.dev), `code`.\n\n" +
	"- one\n- two\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n```go\nfunc main() {}\n```\n\nLast [bad](javascript:alert(1)) one.\n"

func TestViewShows(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) { View(c, sample, nil) }, 600, 800)
	for _, s := range []string{"Title", "Some bold and a link, code.", "one", "two", "a", "b", "1", "2", "func main() {}", "Copy", "go"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func TestViewLinks(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) { View(c, sample, nil) }, 600, 800)
	if err := tt.Click("link"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("bad"); err != nil {
		t.Fatal(err)
	}
	if got := tt.OpenedURLs(); !slices.Equal(got, []string{"https://go.dev"}) {
		t.Errorf("opened %q", got)
	}
	var clicked []string
	o := &Options{OnLink: func(url string) { clicked = append(clicked, url) }}
	tt = ui.NewTester(func(c *ui.Context) { View(c, sample, o) }, 600, 800)
	tt.Click("bad")
	if !slices.Equal(clicked, []string{"javascript:alert(1)"}) || len(tt.OpenedURLs()) > 0 {
		t.Errorf("OnLink got %q, opened %q", clicked, tt.OpenedURLs())
	}
}

// TestViewTableHeaderLinks reaches a link in a table's header, which
// shows what is inside it as a cell does, still named by its text.
func TestViewTableHeaderLinks(t *testing.T) {
	src := "| [Name](https://go.dev) | Size |\n|---|---|\n| a.txt | 1 KB |\n"
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, nil) }, 600, 400)
	for _, name := range []string{"Name", "Size"} {
		if _, ok := tt.AccessByRole(ui.RoleColumnHeader, name); !ok {
			t.Errorf("no column header named %q", name)
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleLink, "Name"); !ok {
		t.Error("the header's link is not in the tree")
	}
}

func TestViewCopyButton(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) { View(c, sample, nil) }, 600, 800)
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != "func main() {}\n" {
		t.Errorf("copied %q", got)
	}
	if !tt.HasText("Copied") {
		t.Errorf("no Copied in %q", tt.Texts())
	}
}

func TestViewCopySelection(t *testing.T) {
	src := "# Title\n\nFirst **para**.\n\n- one\n- two\n\n| a | b |\n|---|---|\n| 1 | 2 |\n"
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, nil) }, 600, 800)
	r, _ := tt.Find("First para.")
	tt.ClickAt(r.X+2, r.Y+r.H/2)
	tt.Command("selectAll")
	tt.Command("copy")
	want := "Title\n\nFirst para.\n\none\ntwo\n\na\tb\n1\t2"
	if got := tt.Clipboard(); got != want {
		t.Errorf("copied %q, want %q", got, want)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyC)
	if got := tt.Clipboard(); got != strings.TrimSuffix(src, "\n") {
		t.Errorf("copied as Markdown %q, want %q", got, src)
	}
}

func TestViewCopyMarkdownPartial(t *testing.T) {
	src := "Some **bold text**"
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, nil) }, 600, 400)
	r, _ := tt.Find("Some bold text")
	w := float32(50) // within "bold", in the system font of 14 DIPs
	// Drag from inside "bold" past the end.
	y := r.Y + r.H/2
	tt.Press(r.X+w, y)
	tt.Move(r.X+r.W-2, y)
	tt.Release(r.X+r.W-2, y)
	tt.Key(ui.Cmd|ui.Shift, ui.KeyC)
	got := tt.Clipboard()
	if !strings.HasPrefix(got, "**") || !strings.HasSuffix(got, "text**") || strings.Contains(got, "Some") {
		t.Errorf("copied as Markdown %q, want the end of the strong text, completed", got)
	}
}

// testdata returns a document of the parser's tests.
func testdata(t testing.TB, name string) string {
	b, err := os.ReadFile("parser/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestStreamMatchesView streams a document a character at a time into
// Stream and checks it shows as View shows it whole.
func TestStreamMatchesView(t *testing.T) {
	src := testdata(t, "extras.md")
	o := &Options{Reveal: &Reveal{Unit: AtOnce}}
	var text string
	done := false
	tt := ui.NewTester(func(c *ui.Context) {
		if done {
			View(c, text, o)
		} else {
			Stream(c, text, o)
		}
	}, 700, 3000)
	for i := 0; i < len(src); {
		_, n := utf8.DecodeRuneInString(src[i:])
		i += n
		text = src[:i]
		if i%7 == 0 {
			tt.Frame()
		}
	}
	done = true
	tt.Frame()
	whole := ui.NewTester(func(c *ui.Context) { View(c, src, o) }, 700, 3000)
	if got, want := tt.Texts(), whole.Texts(); !slices.Equal(got, want) {
		t.Errorf("streamed shows\n%q\nwhole shows\n%q", got, want)
	}
}

// TestRevealPaces checks that text streamed in a burst comes in over
// frames, and all of it once the stream is done.
func TestRevealPaces(t *testing.T) {
	text := ""
	done := false
	o := &Options{}
	tt := ui.NewTester(func(c *ui.Context) {
		if done {
			View(c, text, o)
		} else {
			Stream(c, text, o)
		}
	}, 600, 400)
	text = strings.Repeat("word ", 200)
	tt.Frame()
	first := strings.Join(tt.Texts(), "")
	if len(first) >= len(text)-1 {
		t.Fatalf("a burst showed at once: %d of %d bytes", len(first), len(text))
	}
	tt.Advance(60 * time.Millisecond)
	if second := strings.Join(tt.Texts(), ""); len(second) <= len(first) {
		t.Fatalf("no more text after a while: %d then %d bytes", len(first), len(second))
	}
	done = true
	// Within two seconds of the clock.
	for range 100 {
		tt.Advance(20 * time.Millisecond)
		if strings.Join(tt.Texts(), "") == strings.TrimSpace(text) {
			return
		}
	}
	t.Errorf("not all text showed: %d of %d bytes", len(strings.Join(tt.Texts(), "")), len(text))
}

// TestSelectionSurvivesStreaming selects text of a reply as more of it
// streams in.
func TestSelectionSurvivesStreaming(t *testing.T) {
	text := "First paragraph here.\n\nSecond one is being"
	o := &Options{Reveal: &Reveal{Unit: AtOnce}}
	tt := ui.NewTester(func(c *ui.Context) { Stream(c, text, o) }, 600, 400)
	r, _ := tt.Find("First paragraph here.")
	tt.ClickAt(r.X+2, r.Y+r.H/2)
	tt.Command("selectAll")
	text += " written, word by word."
	tt.Frame()
	tt.Command("copy")
	if got := tt.Clipboard(); !strings.HasPrefix(got, "First paragraph here.\n\nSecond one is being") {
		t.Errorf("copied %q after streaming", got)
	}
}

func TestThemeOverrides(t *testing.T) {
	th := &Theme{Link: ui.RGB(255, 0, 0), FontSize: 20}
	def := DefaultTheme(ui.LightTheme())
	got := resolve(th, def)
	if got.Link != th.Link || got.FontSize != 20 || got.Text != def.Text || got.Headings[0].Size != def.Headings[0].Size {
		t.Errorf("resolved %+v", got)
	}
}

// TestThemeLinkSpans checks that Theme.LinkWeight lifts link text, code
// in a link too, which keeps the code face and wash, and leaves text
// outside links alone.
func TestThemeLinkSpans(t *testing.T) {
	o := &Options{Theme: &Theme{LinkWeight: 500}}
	th := resolve(o.Theme, DefaultTheme(ui.LightTheme()))
	var st *viewState
	ui.NewTester(func(c *ui.Context) {
		st = *ui.Local(View(c, "Read [the `x.go` docs](https://example.com) and `y.go`.", o), stateKey{}, newState)
	}, 400, 100)
	var link, linkCode, code *ui.Span
	for i := range st.paras {
		for _, pc := range st.paras[i].p.pieces {
			for k := range pc.spans {
				switch s := &pc.spans[k]; s.Text {
				case "the ":
					link = s
				case "x.go":
					linkCode = s
				case "y.go":
					code = s
				}
			}
		}
	}
	if link == nil || linkCode == nil || code == nil {
		t.Fatalf("link or code span missing in %+v", st.paras)
	}
	if link.Weight != th.LinkWeight {
		t.Errorf("link text weighs %d, want %d", link.Weight, th.LinkWeight)
	}
	if linkCode.Font != th.CodeFont || linkCode.Background != th.CodeBackground || linkCode.Weight != th.LinkWeight {
		t.Errorf("code in a link shows %+v, want the code face and wash at %d", *linkCode, th.LinkWeight)
	}
	if code.Font != th.CodeFont || code.Background != th.CodeBackground || code.Weight == th.LinkWeight {
		t.Errorf("code outside a link shows %+v, want the code face and wash at its own weight", *code)
	}
}

// TestThemeLink checks that links draw in Theme.Link, not the window
// theme's accent.
func TestThemeLink(t *testing.T) {
	o := &Options{Theme: &Theme{Link: ui.RGB(255, 0, 0)}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, "plain [a link](https://example.com)", o) }, 300, 80)
	img := tt.Image()
	acc := ui.LightTheme().Accent
	reds, accent := 0, 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r, g, b = r>>8, g>>8, b>>8
			if r > 200 && g < 60 && b < 60 {
				reds++
			}
			if r == uint32(acc.R) && g == uint32(acc.G) && b == uint32(acc.B) {
				accent++
			}
		}
	}
	if reds == 0 || accent > 0 {
		t.Errorf("%d pixels in Theme.Link, %d in the window theme's accent", reds, accent)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"Hello, World!": "hello-world", "API v2 — notes": "api-v2--notes", "foo_bar-baz": "foo_bar-baz"} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestFadeOrder checks that text revealed later is more transparent.
func TestFadeOrder(t *testing.T) {
	now := time.Now()
	rs := revealState{marks: []revealMark{{0, now.Add(-250 * time.Millisecond)}, {5, now.Add(-150 * time.Millisecond)}, {8, now.Add(-20 * time.Millisecond)}}}
	f := rs.fadeAt(now, false, nil)
	a, b, c := f(1), f(6), f(9)
	if !(a > b && b > c) {
		t.Errorf("opacities %v %v %v, want decreasing", a, b, c)
	}
	if f(100) != c {
		t.Errorf("text after the last mark: %v", f(100))
	}
}

// TestFadeReplayed streams anew, a paragraph at a time, text the view
// showed whole, and checks that the text fades in all the same.
func TestFadeReplayed(t *testing.T) {
	const src = "First paragraph here.\n\n1. An item\n2. Another\n\nLast one.\n"
	text, done := src, true
	o := &Options{Reveal: &Reveal{Unit: ByParagraph}}
	var st *viewState
	tt := ui.NewTester(func(c *ui.Context) {
		var root ui.Element
		if done {
			root = View(c, text, o)
		} else {
			root = Stream(c, text, o)
		}
		st = *ui.Local(root, stateKey{}, newState)
	}, 600, 400)
	text, done = "", false
	tt.Frame()
	text = src
	tt.Frame()
	faded := false
	for range 20 {
		tt.Advance(10 * time.Millisecond)
		for i := range st.paras {
			for _, pc := range st.paras[i].p.pieces {
				for _, s := range pc.spans {
					faded = faded || s.Color.A > 0 && s.Color.A < 200
				}
			}
		}
	}
	if !faded {
		t.Error("text streamed again did not fade in")
	}
}

func TestSentenceEnds(t *testing.T) {
	for _, c := range []struct {
		src  string
		want int
	}{
		{"Pi is 3.14 today. Next", len("Pi is 3.14 today. ")},
		{`He said "go." Then`, len(`He said "go." `)},
		{"Really? Yes", len("Really? ")},
		{"一句话。下一句", len("一句话。")},
		{"- an item\n- next", len("- an item\n")},
		{"Not done yet", -1},
		{"Ends here.", -1}, // the space after it has not come
	} {
		if got := unitEnd(c.src, 0, len(c.src), BySentence); got != c.want {
			t.Errorf("%q: sentence ends at %d, want %d", c.src, got, c.want)
		}
	}
}

// TestRevealBySentence checks that Stream by sentence shows only the
// sentences complete while more is coming, and all once done.
func TestSnapByWordCJK(t *testing.T) {
	src := "Go语言 is 今日は。"
	for _, c := range []struct{ i, want int }{
		{0, 2},  // "Go", up to 语
		{1, 2},  // in "Go"
		{2, 5},  // 语
		{3, 5},  // inside 语
		{5, 9},  // 言 and the space after it
		{9, 12}, // "is "
		{12, 15},
		{15, 18},
		{18, 21},
		{21, 24}, // 。
	} {
		if got := snap(src, c.i, len(src), ByWord); got != c.want {
			t.Errorf("snap(%d) = %d, want %d", c.i, got, c.want)
		}
	}
}

// TestRevealCJKByWord streams Chinese in bursts, as a model's replies
// come over a network: ByWord shows it a character or two at a time, not
// each burst at once, as it shows English a word at a time (#668).
func TestRevealCJKByWord(t *testing.T) {
	zh := []rune(strings.Repeat("今天天气很好，我们去公园散步，看到了很多鸟在树上唱歌。", 4))
	const burst = 30
	text := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(10).Children(func() { Stream(c, text, &Options{}) })
	}, 600, 400)
	tt.Frame()
	shown, steps, biggest := 0, 0, 0
	for fed, frame := 0, 0; shown < len(zh) && frame < 600; frame++ {
		if frame%12 == 0 && fed < len(zh) { // a burst every 12 frames
			fed = min(fed+burst, len(zh))
			text = string(zh[:fed])
			tt.Frame()
		}
		tt.Advance(16 * time.Millisecond)
		if n := utf8.RuneCountInString(strings.Join(tt.Texts(), "")); n != shown {
			steps, biggest, shown = steps+1, max(biggest, n-shown), n
		}
	}
	if shown != len(zh) {
		t.Fatalf("showed %d of %d runes", shown, len(zh))
	}
	t.Logf("%d visible changes, the largest %d runes", steps, biggest)
	if biggest > 4 || steps < len(zh)/4 {
		t.Errorf("%d visible changes, the largest %d runes: CJK came in bursts", steps, biggest)
	}
}

func TestRevealBySentence(t *testing.T) {
	text, done := "", false
	o := &Options{Reveal: &Reveal{Unit: BySentence, CatchUp: time.Millisecond}}
	tt := ui.NewTester(func(c *ui.Context) {
		if done {
			View(c, text, o)
		} else {
			Stream(c, text, o)
		}
	}, 600, 400)
	text = "First sentence here. Second one is still"
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	if got := strings.Join(tt.Texts(), ""); got != "First sentence here." {
		t.Errorf("streaming shows %q", got)
	}
	done = true
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	if got := strings.Join(tt.Texts(), ""); got != text {
		t.Errorf("done, shows %q", got)
	}
}
