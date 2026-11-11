//go:build darwin

package darwin

import "testing"

// TestStringOf reads text from strings and attributed strings, and from the
// NSNumber VoiceOver sends as a text field's AXValue (issue #802), which
// has no UTF8String.
func TestStringOf(t *testing.T) {
	load()
	withPool(func() {
		attributed := autorelease(send(send(class("NSAttributedString"), "alloc"), "initWithString:", uintptr(nsString("rich"))))
		for name, c := range map[string]struct {
			obj  id
			want string
		}{
			"string":     {nsString("plain"), "plain"},
			"attributed": {attributed, "rich"},
			"number":     {nsNumberInt(42), "42"},
			"null":       {send(class("NSNull"), "null"), ""},
			"nil":        {0, ""},
		} {
			if got := stringOf(c.obj); got != c.want {
				t.Errorf("stringOf(%s) = %q, want %q", name, got, c.want)
			}
		}
	})
}
