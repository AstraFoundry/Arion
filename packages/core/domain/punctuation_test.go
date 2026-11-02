package domain

import "testing"

func TestApplyPunctuation(t *testing.T) {
	const marked = "你好，世界。"
	cases := []struct {
		name string
		mode Punctuation
		in   string
		want string
	}{
		{"smart keeps model punctuation", PunctuationSmart, marked, marked},
		{"full keeps model punctuation", PunctuationFull, marked, marked},
		{"empty keeps model punctuation", "", marked, marked},
		{"omit drops one ideographic period", PunctuationOmit, marked, "你好，世界"},
		{"omit drops one question mark", PunctuationOmit, "真的吗？", "真的吗"},
		{"omit drops one exclamation", PunctuationOmit, "太好了！", "太好了"},
		{"omit drops one ellipsis", PunctuationOmit, "未完…", "未完"},
		{"omit drops one ascii period", PunctuationOmit, "Wait.", "Wait"},
		{"omit drops one ascii question", PunctuationOmit, "Ready?", "Ready"},
		{"omit drops one ascii exclamation", PunctuationOmit, "Wait!", "Wait"},
		{"omit drops repeated periods", PunctuationOmit, "好。。", "好"},
		{"omit drops a full ellipsis", PunctuationOmit, "未完……", "未完"},
		{"omit drops ascii ellipsis", PunctuationOmit, "Wait...", "Wait"},
		{"omit drops mixed final marks", PunctuationOmit, "真的吗？！", "真的吗"},
		{"omit keeps a trailing comma", PunctuationOmit, "你好，", "你好，"},
		{"omit keeps an internal mark", PunctuationOmit, "好。的", "好。的"},
		{"omit handles a trailing space", PunctuationOmit, "好。 ", "好 "},
		{"omit handles trailing unicode whitespace", PunctuationOmit, "好。\t\r\n\u3000", "好\t\r\n\u3000"},
		{"omit preserves earlier sentence marks", PunctuationOmit, "你好！世界。\n", "你好！世界\n"},
		{"omit preserves unpunctuated whitespace", PunctuationOmit, "好 \n", "好 \n"},
		{"omit handles empty input", PunctuationOmit, "", ""},
		{"omit handles whitespace only", PunctuationOmit, " \n", " \n"},
		{"omit handles marks only", PunctuationOmit, "……\n", "\n"},
		{"spaces replaces punctuation and collapses spaces", PunctuationSpaces, marked, "你好 世界 "},
		{"spaces collapses adjacent marks", PunctuationSpaces, "a!!  b", "a b"},
		{"spaces replaces an apostrophe", PunctuationSpaces, "don't", "don t"},
		{"spaces leaves letters", PunctuationSpaces, "hello", "hello"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ApplyPunctuation(c.mode, c.in); got != c.want {
				t.Fatalf("ApplyPunctuation(%q, %q) = %q, want %q", c.mode, c.in, got, c.want)
			}
		})
	}
}

func TestPunctuationKnown(t *testing.T) {
	for _, mode := range []Punctuation{PunctuationSmart, PunctuationFull, PunctuationOmit, PunctuationSpaces} {
		if !mode.Known() {
			t.Fatalf("%q unknown", mode)
		}
	}
	if Punctuation("comma").Known() || Punctuation("").Known() {
		t.Fatal("invalid mode accepted")
	}
}
