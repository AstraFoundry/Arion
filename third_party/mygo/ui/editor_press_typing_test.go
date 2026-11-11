package ui

import "testing"

// Typing while the button is still down, as a touchpad's late release lets it,
// must not move the caret back to the pointer between the keys.
func TestTypingEndsTheDragOfAPress(t *testing.T) {
	for _, tc := range []struct {
		name   string
		clicks int
		x      float32
		want   string
	}{
		{"caret at the start", 1, 2, "8085"},
		{"caret at the end", 1, 150, "8580"},
		{"a word selected", 2, 20, "80"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := "85"
			tt := NewTester(func(c *Context) {
				TextInput(c, &value).Width(200)
			}, 300, 100)
			x, y := tc.x, float32(16)
			for range tc.clicks - 1 {
				tt.Press(x, y)
				tt.Release(x, y)
			}
			tt.Press(x, y)
			tt.Frame()
			tt.TypeKey(0, Key8, "8")
			tt.Frame()
			tt.TypeKey(0, Key0, "0")
			tt.Frame()
			tt.Release(x, y)
			tt.Frame()
			if value != tc.want {
				t.Errorf("value %q, want %q", value, tc.want)
			}
		})
	}
}
