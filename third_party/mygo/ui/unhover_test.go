package ui

import "testing"

// TestHoverAsPopupCloses checks that a select's trigger, the pointer
// resting on it, hovers in the frame its popup closes, as the popup's
// backdrop covered it: it did not until the next frame, so a trigger
// lit while open or hovered went dark for a frame.
func TestHoverAsPopupCloses(t *testing.T) {
	for _, how := range []string{"click", "escape"} {
		t.Run(how, func(t *testing.T) {
			choice := 1
			var tt *Tester
			lit := map[uint64]bool{} // each frame's last pass
			tt = coreNewTester(func(c *context) {
				coreColumn(c).Padding(20).Children(func() {
					s := coreSelectBase(c, &choice)
					s.Trigger.Size(80, 24).Label("Pick")
					lit[c.rt.frame] = s.Trigger.Hovered() || s.Open()
					s.Popup(func(panel *node) {
						panel.Width(120)
						for v := range 3 {
							s.Item(v).Height(24).Label(string(rune('A' + v)))
						}
					})
				})
			}, 300, 200)
			tt.Frame()
			tt.Move(40, 32)
			tt.Frame()
			tt.ClickAt(40, 32)
			tt.Frame()
			if !lit[tt.rt.frame] {
				t.Fatal("the popup did not open")
			}
			from := tt.rt.frame
			if how == "click" {
				tt.ClickAt(40, 32) // on the backdrop, over the trigger
			} else {
				tt.Key(0, KeyEscape)
			}
			tt.Frame()
			for f := from; f <= tt.rt.frame; f++ {
				if on, ok := lit[f]; ok && !on {
					t.Fatalf("frame %d: the trigger under the pointer went dark", f)
				}
			}
		})
	}
}
