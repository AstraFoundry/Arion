//go:build darwin

package darwin

import "testing"

// PingFang SC and Songti SC come with macOS. Songti SC of macOS 26 also
// draws ⼜ (U+2F1C), the radical of 又, with the glyph of 口; the PDF tests
// of package mygo cover what that does to printed text.
func TestRadicalIdeographs(t *testing.T) {
	for font, want := range map[string]map[rune]rune{
		"PingFangSC-Regular":  {'⼈': '人', '⼝': '口', '⼜': '又'},
		"STSongti-SC-Regular": {'⼈': '人', '⼝': '口'},
	} {
		got := RadicalIdeographs(font, []rune{'⼈', '⼝', '⼜'})
		for r, i := range want {
			if got[r] != i {
				t.Errorf("%s: %c → %q, want %c", font, r, got[r], i)
			}
		}
	}
	if got := RadicalIdeographs("NoSuchFont-Regular", []rune{'⼈'}); got != nil {
		t.Errorf("a font that is not installed gave %q", got)
	}
}
