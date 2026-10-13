// Package stepper draws onboarding progress as dots that stretch into a
// pill for the current step.
package stepper

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Render draws total dots with index current active.
func Render(c *ui.Context, tokens theme.Tokens, current, total int) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Label(copy.Step(current+1, total+1)).Children(func() {
		for i := 0; i <= total; i++ {
			dot := ui.Box(c.Key(i)).Height(6).Radius(3)
			width := float32(6)
			color := tokens.CardBorder
			switch {
			case i == current:
				width, color = 22, tokens.Accent
			case i < current:
				color = tokens.Accent.Alpha(0.45)
			}
			dot.Width(dot.AnimateTo("w", width, theme.Settle)).Background(dot.AnimateColor("c", color, theme.Quick))
		}
	})
}
