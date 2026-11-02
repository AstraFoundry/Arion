package domain

import (
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

func TestKeyboardContextKeepsNearbyText(t *testing.T) {
	c := RewriteContext{Before: strings.Repeat("旧", 200) + "最近的前文", After: "最近的后文" + strings.Repeat("远", 200), Screen: strings.Repeat("屏", 4000) + "窗口末尾", History: []string{"voice history"}}
	out, selected := c.KeyboardBounded()
	if !selected || !strings.HasSuffix(out.Before, "最近的前文") || !strings.HasPrefix(out.After, "最近的后文") || !strings.HasSuffix(out.Screen, "窗口末尾") || len(out.History) != 0 {
		t.Fatal("keyboard selection lost nearby text or retained speech history")
	}
	for text, want := range map[string]int{out.Before: MaxKeyboardCursorContextUnits, out.After: MaxKeyboardCursorContextUnits, out.Screen: MaxKeyboardScreenContextUnits} {
		if got := len(utf16.Encode([]rune(text))); got != want {
			t.Fatalf("selected %d UTF-16 units, want %d", got, want)
		}
	}
	voice := c.Bounded()
	if voice.Screen != c.Screen {
		t.Fatal("keyboard policy changed complete-window dictation")
	}
}

func TestKeyboardContextPreservesSurrogateBoundaries(t *testing.T) {
	for _, suffix := range []bool{true, false} {
		for _, text := range []string{"甲😀乙", "😀😀😀", "甲乙丙", ""} {
			out := keyboardContextSlice(text, 3, suffix)
			if !utf8.ValidString(out) || len(utf16.Encode([]rune(out))) > 3 || strings.ContainsRune(out, utf8.RuneError) {
				t.Fatalf("invalid UTF-16 slice %q", out)
			}
			if suffix && !strings.HasSuffix(text, out) || !suffix && !strings.HasPrefix(text, out) {
				t.Fatal("selection changed original text")
			}
		}
	}
	if keyboardContextSlice("甲😀乙", 2, false) != "甲" || keyboardContextSlice("甲😀乙", 2, true) != "乙" {
		t.Fatal("partial surrogate was retained")
	}
	if _, selected := (RewriteContext{Screen: "small"}).KeyboardBounded(); selected {
		t.Fatal("small screen falsely marked selected")
	}
}
