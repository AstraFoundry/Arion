package domain

import "unicode/utf8"

const (
	MaxKeyboardCursorContextUnits = 128
	MaxKeyboardScreenContextUnits = 1024
)

// KeyboardBounded selects nearby caret text and a bounded AX-order window tail.
// Its boolean reports screen selection, not an Accessibility read failure.
// Dictation keeps its separate complete-window context policy.
func (c RewriteContext) KeyboardBounded() (RewriteContext, bool) {
	screen := sanitizeContext(c.Screen)
	out := RewriteContext{
		Before: keyboardContextSlice(sanitizeContext(c.Before), MaxKeyboardCursorContextUnits, true),
		After:  keyboardContextSlice(sanitizeContext(c.After), MaxKeyboardCursorContextUnits, false),
		Screen: keyboardContextSlice(screen, MaxKeyboardScreenContextUnits, true),
	}
	return out, out.Screen != screen
}

// Slice on scalar boundaries while counting the host's UTF-16 string units.
func keyboardContextSlice(s string, limit int, suffix bool) string {
	units := 0
	if suffix {
		for end := len(s); end > 0; {
			r, size := utf8.DecodeLastRuneInString(s[:end])
			units++
			if r > 0xffff {
				units++
			}
			if units > limit {
				return s[end:]
			}
			end -= size
		}
		return s
	}
	for i, r := range s {
		units++
		if r > 0xffff {
			units++
		}
		if units > limit {
			return s[:i]
		}
	}
	return s
}
