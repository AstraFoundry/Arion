//go:build linux && (amd64 || arm64)

package text

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

// Punctuation and full-width forms in a language that writes Han get the
// family the language picks before the generic families (#595), and
// nothing else does.
func TestHanPunctRuns(t *testing.T) {
	for _, c := range []struct {
		text  string
		style Style
		spans []Span
		want  []hanRun
	}{
		{"macOS、Windows、Linux", Style{Lang: "zh-Hans"}, nil, []hanRun{{5, 6, "zh-Hans", ""}, {13, 14, "zh-Hans", ""}}},
		{"a（b）c", Style{Lang: "ja", Family: "Inter"}, nil, []hanRun{{1, 2, "ja", "Inter"}, {3, 4, "ja", "Inter"}}},
		{"文件、。编辑", Style{Lang: "zh"}, nil, []hanRun{{2, 4, "zh", ""}}},
		{"macOS、Windows", Style{Lang: "en"}, nil, nil},
		{"macOS、Windows", Style{}, nil, nil},
		{"a、b、c", Style{Lang: "en"}, []Span{{End: 2, Lang: "zh-Hant"}, {End: 5}}, []hanRun{{1, 2, "zh-Hant", ""}}},
		{"a、、b", Style{Lang: "zh-Hans"}, []Span{{End: 2, Family: "serif"}, {End: 4}}, []hanRun{{1, 2, "zh-Hans", "serif"}, {2, 3, "zh-Hans", ""}}},
	} {
		if got := hanPunctRuns([]rune(c.text), c.style, c.spans); !reflect.DeepEqual(got, c.want) {
			t.Errorf("hanPunctRuns(%q, %+v, %+v) = %v, want %v", c.text, c.style, c.spans, got, c.want)
		}
	}
}

func TestPangoFamilyBeforeGeneric(t *testing.T) {
	before := func(generic string) string { return "CJK " + generic }
	for _, c := range []struct{ list, want string }{
		{"", "CJK system-ui,UI,system-ui,sans-serif"},
		{"sans-serif", "CJK sans-serif,sans-serif"},
		{"Inter, monospace, serif", "Inter,CJK monospace,monospace,serif"},
		{"Inter", "Inter,CJK system-ui"},
	} {
		if got := pangoFamily(c.list, "UI", before); got != c.want {
			t.Errorf("pangoFamily(%q) = %q, want %q", c.list, got, c.want)
		}
		if got, want := pangoFamily(c.list, "UI", func(string) string { return "" }), pangoFamily(c.list, "UI", nil); got != want {
			t.Errorf("pangoFamily(%q) without a family = %q, want %q", c.list, got, want)
		}
	}
}

func TestFcHanLang(t *testing.T) {
	for tag, want := range map[string]string{
		"zh": "zh-cn", "zh-Hans": "zh-cn", "zh-CN": "zh-cn", "zh_CN": "zh-cn", "zh-Hans-SG": "zh-sg", "zh-SG": "zh-sg",
		"zh-Hant": "zh-tw", "zh-TW": "zh-tw", "zh-Hant-HK": "zh-hk", "zh-HK": "zh-hk", "zh-MO": "zh-mo", "zh-Hans-HK": "zh-cn",
		"yue": "zh-hk", "yue-Hans": "zh-cn", "cmn-Hant-TW": "zh-tw", "ja": "ja", "ja-JP": "ja", "ko-KR": "ko",
	} {
		if got := fcHanLang(tag); got != want {
			t.Errorf("fcHanLang(%q) = %q, want %q", tag, got, want)
		}
	}
}

// With the system's fonts, an ideographic comma between Latin words in
// Chinese is drawn with the font fontconfig picks for Chinese (#595).
func TestPangoHanPunctBetweenLatin(t *testing.T) {
	e, err := newPango()
	if err != nil {
		t.Skip(err)
	}
	han := e.hanFamily("zh-Hans", "sans-serif")
	if han == "" {
		t.Skip("no font for Chinese")
	}
	want := e.shape([]rune("、"), Style{Family: han, Size: 12, Lang: "zh-Hans"}, nil, 0, false, false)[0].runs[0].font
	text := []rune("macOS、Windows")
	for _, r := range e.shape(text, Style{Family: "sans-serif", Size: 12, Lang: "zh-Hans"}, nil, 0, false, false)[0].runs {
		if r.start <= 5 && 5 < r.end && r.font != want {
			t.Errorf("、 drawn with another font than %s's", han)
		}
	}
}

// A flex item may give text exactly the width of its unconstrained
// layout. Kerning in strings such as "11" must not make Pango wrap it.
func TestPangoLayoutAtMeasuredWidth(t *testing.T) {
	s := newSystem()
	for _, family := range []string{"sans-serif", "serif", "monospace"} {
		for _, size := range []float32{10, 12, 14, 16} {
			for _, value := range []string{"11", "10", "12", "1911", "2011", "2111", "2211", "2001", "2002", "2012", "2021", "2011-04-17", "2011-06-17", "11:30", "2.11.1", "AV", "office"} {
				t.Run(fmt.Sprintf("%s/%g/%s", family, size, value), func(t *testing.T) {
					p := Params{Text: value, Style: Style{Family: family, Size: size}}
					p.Width = s.Layout(p).Width
					for _, maxLines := range []int{0, 1} {
						p.MaxLines = maxLines
						l := s.Layout(p)
						if len(l.Lines) != 1 || l.Truncated || l.Lines[0].End != len(l.Runes) {
							t.Errorf("at measured width %g, MaxLines %d: %d lines, truncated %v, end %d of %d", p.Width, maxLines, len(l.Lines), l.Truncated, l.Lines[0].End, len(l.Runes))
						}
					}
				})
			}
		}
	}
}

// Letter spacing makes a text's measured width fractional, so a flex
// item may lay it out at a width a thousandth of a dip short of its own
// measurement. Truncating the scaled width must not wrap it.
func TestPangoLayoutAtMeasuredWidthWithLetterSpacing(t *testing.T) {
	s := newSystem()
	for _, spacing := range []float32{0.1, 0.25, 0.5, 1, 1.5} {
		for _, value := range []string{"System", "Settings", "Favorites", "Home", "Now Playing"} {
			t.Run(fmt.Sprintf("%g/%s", spacing, value), func(t *testing.T) {
				p := Params{Text: value, Style: Style{Family: "sans-serif", Size: 14, LetterSpacing: spacing}}
				p.Width = s.Layout(p).Width
				for _, maxLines := range []int{0, 1} {
					p.MaxLines = maxLines
					l := s.Layout(p)
					if len(l.Lines) != 1 || l.Truncated || l.Lines[0].End != len(l.Runes) {
						t.Errorf("at measured width %g, MaxLines %d: %d lines, truncated %v, end %d of %d", p.Width, maxLines, len(l.Lines), l.Truncated, l.Lines[0].End, len(l.Runes))
					}
				}
			})
		}
	}
}

// Goroutines that write the same font to the cache at once each find the
// whole font there afterwards (#682).
func TestWriteCacheFileConcurrent(t *testing.T) {
	data := make([]byte, 1<<20)
	for i := range data {
		data[i] = byte(i * 7)
	}
	for round := range 20 {
		dir := t.TempDir()
		path := filepath.Join(dir, "font.font")
		var wg sync.WaitGroup
		errs := make([]error, 16)
		for i := range errs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := writeCacheFile(dir, path, data); err != nil {
					errs[i] = err
					return
				}
				got, err := os.ReadFile(path)
				if err == nil && !bytes.Equal(got, data) {
					err = fmt.Errorf("read %d bytes, not the font's %d", len(got), len(data))
				}
				errs[i] = err
			}()
		}
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d, writer %d: %v", round, i, err)
			}
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("round %d: %d files in the cache, want only the font", round, len(entries))
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if mode := info.Mode().Perm(); mode != 0o644 {
			t.Fatalf("round %d: font's mode %v, want 0644", round, mode)
		}
	}
}

// lcdFilterRule gives subpixel glyphs FreeType's LCD filter where
// fontconfig names no filter, as cairo 1.17 and later do by default, and
// keeps a filter fontconfig names (#1025).
func TestLCDFilterRule(t *testing.T) {
	if _, err := newPango(); err != nil {
		t.Skip(err)
	}
	l := &pangoLib
	if l.cairoVersion() < 11700 || l.fcConfigParseFromMemory == nil {
		t.Skip("needs cairo 1.17 or later, whose default filter is FreeType's")
	}
	draw := func() []byte {
		e, err := newPango()
		if err != nil {
			t.Fatal(err)
		}
		e.setFontRendering("subpixel", "slight", "rgb")
		var out []byte
		for _, r := range e.shape([]rune("Illumination 测试"), Style{Family: "sans-serif", Size: 13}, nil, 0, false, false)[0].runs {
			for _, g := range r.glyphs {
				if b := e.draw(r.font, g.ID, 1, 0, true); b.pix != nil {
					if !b.subpixel {
						t.Fatal("glyph not drawn with subpixel antialiasing")
					}
					out = append(out, b.pix...)
				}
			}
		}
		return out
	}
	// Rules loaded later apply later, as the user's after the system's.
	rule := func(xml string) {
		t.Helper()
		if l.fcConfigParseFromMemory(l.fcConfigGetCurrent(), xml, int32(gTrue)) == 0 {
			t.Fatalf("fontconfig refused %s", xml)
		}
	}
	set := func(edit string) string {
		return `<?xml version="1.0"?><!DOCTYPE fontconfig SYSTEM "fonts.dtd"><fontconfig><match target="font">` + edit + `</match></fontconfig>`
	}
	rule(set(`<edit name="lcdfilter" mode="assign"><const>lcdlegacy</const></edit>`))
	legacy := draw()
	rule(lcdFilterRule)
	if !bytes.Equal(draw(), legacy) {
		t.Error("lcdFilterRule replaced the legacy filter fontconfig names")
	}
	rule(set(`<edit name="lcdfilter" mode="delete"/>`))
	unnamed := draw()
	if bytes.Equal(unnamed, legacy) {
		t.Fatal("cairo draws subpixel glyphs with the legacy filter by default")
	}
	rule(lcdFilterRule)
	if !bytes.Equal(draw(), unnamed) {
		t.Error("where fontconfig names no filter, lcdFilterRule draws otherwise than cairo's default")
	}
	rule(set(`<edit name="lcdfilter" mode="assign"><const>lcddefault</const></edit>`))
	if !bytes.Equal(draw(), unnamed) {
		t.Error("cairo's default filter is not FreeType's")
	}
}
