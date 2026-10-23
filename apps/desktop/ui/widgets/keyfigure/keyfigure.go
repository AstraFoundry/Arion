// Package keyfigure illustrates the dictation key: an isometric keyboard
// on a desk mat, wired to a microphone, with the chosen key lit. A press
// sinks the cap and lights the microphone's ring; choosing another key
// moves the light to it. The layers are drawn by
// scripts/build-holdkey-figures.mjs with the Anatomy kit.
package keyfigure

import (
	"slices"
	"time"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// platforms have a keyboard each in the figure.
var platforms = []string{"darwin", "windows"}

// keyboardOf is the platform whose keyboard carries key.
func keyboardOf(key runtime.HoldKey) string {
	for _, platform := range platforms {
		if slices.Contains(runtime.HoldKeysFor(platform), key) {
			return platform
		}
	}
	return ""
}

// Shows reports whether the figure has key, so callers keep another
// illustration for a platform without one.
func Shows(key runtime.HoldKey) bool { return keyboardOf(key) != "" }

// Render fills its box with the figure. Pressed is when the key last went
// down, or zero before it is pressed: each press plays once.
func Render(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, key runtime.HoldKey, pressed time.Time) {
	appearance := "light"
	if tokens.Dark {
		appearance = "dark"
	}
	base := assets.HoldKeyFigure(keyboardOf(key) + "-" + appearance)
	w, h := base.Size()
	// The figure's paper is the card color: it has no background of its own.
	ui.Column(c).Fill().Background(tokens.Card).Justify(ui.Center).AlignItems(ui.Center).
		Role(ui.RoleImage).Label(messages.Text(copy.HoldKeyFigureLabel)).Children(func() {
		ui.Box(c).Size(w, h).Children(func() {
			ui.Image(c, base).Absolute().Top(0).Left(0).Size(w, h)
			// The chosen key crossfades when another is chosen.
			ui.Box(c.Key(string(key))).Absolute().Top(0).Left(0).Size(w, h).
				Transition(ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{}, Exit: &ui.Motion{}}).
				Children(func() {
					ui.Image(c, assets.HoldKeyFigure(string(key)+"-"+appearance)).Absolute().Top(0).Left(0).Size(w, h)
					down := ui.Image(c, assets.HoldKeyFigure(string(key)+"-"+appearance+"-pressed")).Absolute().Top(0).Left(0).Size(w, h)
					// Down quickly, held long enough to see the ring, back up softly.
					var press float32
					if !pressed.IsZero() {
						press = down.Keyframes("press", []ui.Keyframe{
							{Value: 0},
							{At: 90 * time.Millisecond, Value: 1, Ease: ui.EaseOut},
							{At: 700 * time.Millisecond, Value: 1},
							{At: 960 * time.Millisecond, Value: 0, Ease: ui.EaseOut},
						}, ui.KeyframeOptions{Run: uint64(pressed.UnixNano())})
					}
					down.Opacity(press)
				})
		})
	})
}
