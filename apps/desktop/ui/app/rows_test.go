package app

import (
	"testing"

	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Settings rows are one height whatever control they hold, so a button
// row and a switch row in one card line up.
func TestSettingsRowsShareHeight(t *testing.T) {
	on, chosen, selected := true, "One", 0
	titles := []string{"Switch", "Button", "Select", "Segmented", "Link"}
	tt := ui.NewTester(func(c *ui.Context) {
		tokens := theme.Apply(c)
		ui.Column(c).Children(func() {
			kit.TipRow(c, tokens, titles[0], "Tip", func() { ui.Switch(c.Key("switch"), &on) })
			kit.Row(c, tokens, titles[1], "", func() { kit.Button(c, tokens, kit.Outline, "Check", true) })
			kit.Row(c, tokens, titles[2], "", func() { ui.Select(c.Key("select"), &chosen, []string{"One", "Two"}) })
			kit.TipRow(c, tokens, titles[3], "", func() { ui.Segmented(c.Key("segmented"), &selected, "A", "B") })
			kit.LinkRow(c, tokens, titles[4], "Value", nil)
		})
	}, 560, 320)
	settle(tt)
	snapshot(t, tt, "settings-rows")
	// The pitch between row centers is RowHeight. Select and Segmented
	// size from their padding plus the platform text line box, so those
	// rows can exceed the minimum: about 0.6 DIP on Windows and 2 DIP
	// on Linux. macOS fits. A second line of text still fails.
	const rowPitchSlack float32 = 2
	var previous float32
	for i, title := range titles {
		r, ok := tt.Find(title)
		if !ok {
			t.Fatalf("%s row not rendered", title)
		}
		center := r.Y + r.H/2
		if i > 0 {
			pitch := center - previous
			if pitch < theme.RowHeight-rowPitchSlack || pitch > theme.RowHeight+rowPitchSlack {
				t.Errorf("%s row starts %v below the previous row, want %v±%v", title, pitch, theme.RowHeight, rowPitchSlack)
			}
		}
		previous = center
	}
}
