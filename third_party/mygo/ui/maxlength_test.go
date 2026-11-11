package ui

import "testing"

func maxLengthTester(value *string, multiline bool, n int) *Tester {
	return NewTester(func(c *Context) {
		input := TextInput
		if multiline {
			input = TextArea
		}
		input(c, value).MaxLength(n).AutoFocus()
	}, 300, 120)
}

func TestMaxLength(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		// Typing at the limit changes nothing, at the caret too.
		value := "abc"
		tt := maxLengthTester(&value, multiline, 3)
		tt.Key(0, KeyHome)
		tt.Type("X")
		if value != "abc" {
			t.Fatalf("multiline %v: typed into a full input: %q", multiline, value)
		}

		// Text that does not fit is cut to what does, at the caret.
		value = "ab"
		tt = maxLengthTester(&value, multiline, 3)
		tt.Type("XYZ")
		if value != "abX" {
			t.Fatalf("multiline %v: cut to %q", multiline, value)
		}

		// A selection replaced makes room.
		value = "abc"
		tt = maxLengthTester(&value, multiline, 3)
		tt.Key(Cmd, KeyA)
		tt.Type("Z")
		if value != "Z" {
			t.Fatalf("multiline %v: replacing the selection gave %q", multiline, value)
		}
		tt.Key(Cmd, KeyA)
		tt.Type("12345")
		if value != "123" {
			t.Fatalf("multiline %v: replacing the selection with more gave %q", multiline, value)
		}

		// A paste is cut as typing is.
		value = "a"
		tt = maxLengthTester(&value, multiline, 3)
		tt.SetClipboard("hello")
		tt.Key(Cmd, KeyV)
		if value != "ahe" {
			t.Fatalf("multiline %v: pasted %q", multiline, value)
		}

		// An input method's commit is cut; its composition is not.
		value = "a"
		tt = maxLengthTester(&value, multiline, 3)
		tt.Compose("xyz", 3)
		tt.Type("xyz")
		if value != "axy" {
			t.Fatalf("multiline %v: committed %q", multiline, value)
		}

		// Length is in UTF-16 code units, and a surrogate pair is not split.
		value = ""
		tt = maxLengthTester(&value, multiline, 3)
		tt.Type("a😀b")
		if value != "a😀" {
			t.Fatalf("multiline %v: a😀b cut to %q", multiline, value)
		}
		value = "ab"
		tt = maxLengthTester(&value, multiline, 3)
		tt.Type("😀")
		if value != "ab" {
			t.Fatalf("multiline %v: a pair split into %q", multiline, value)
		}

		// A keystroke that does not fit leaves no step to undo.
		value = ""
		tt = maxLengthTester(&value, multiline, 3)
		tt.Type("abc")
		tt.Type("Y")
		tt.Key(Cmd, KeyZ)
		if value != "" {
			t.Fatalf("multiline %v: undone, it became %q", multiline, value)
		}

		// No limit at zero.
		value = "abc"
		tt = maxLengthTester(&value, multiline, 0)
		tt.Type("d")
		if value != "abcd" {
			t.Fatalf("multiline %v: unlimited input became %q", multiline, value)
		}
	}
}

func TestMaxLengthNewline(t *testing.T) {
	value := "ab"
	tt := maxLengthTester(&value, true, 3)
	tt.Key(0, KeyEnter)
	tt.Type("X")
	if value != "ab\n" {
		t.Fatalf("a newline counted as more than one: %q", value)
	}
}

func TestMaxLengthLeavesValueSetByApp(t *testing.T) {
	value := "abc"
	tt := maxLengthTester(&value, false, 3)
	value = "abcdef"
	tt.Frame()
	tt.Key(0, KeyEnd)
	tt.Type("X")
	if value != "abcdef" {
		t.Fatalf("a value the app set became %q", value)
	}
	tt.Key(0, KeyBackspace)
	if value != "abcde" {
		t.Fatalf("Backspace in a value past the limit gave %q", value)
	}
}
