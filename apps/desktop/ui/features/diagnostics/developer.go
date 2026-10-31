package diagnostics

import (
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// DeveloperEntry owns the hidden, window-local unlock gesture.
type DeveloperEntry struct {
	started  time.Time
	taps     int
	unlocked bool
}

func (e *DeveloperEntry) tap(now time.Time) bool {
	if e.started.IsZero() || now.Sub(e.started) > 5*time.Second {
		e.started, e.taps = now, 0
	}
	e.taps++
	if e.taps < 10 {
		return false
	}
	e.taps = 0
	e.unlocked = true
	return true
}

func (e *DeveloperEntry) Version(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	row := ui.Row(c.Key("developer-entry")).Children(func() {
		ui.Text(c, store.Copy().Version(store.State.Version)).FontSize(theme.Caption).TextColor(tokens.Muted)
	})
	if row.Clicked() && e.tap(time.Now()) {
		store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.DeveloperMode = true })
	}
}

func (e *DeveloperEntry) Card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if !e.unlocked && !store.State.Preferences.DeveloperMode {
		return
	}
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, store.Copy().Text(copy.DeveloperMode), store.Copy().Text(copy.DeveloperModeDetail), func() {
			on := store.State.Preferences.DeveloperMode
			if ui.Switch(c.Key("developer-mode"), &on).Label(store.Copy().Text(copy.DeveloperMode)).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.DeveloperMode = on })
			}
		})
	})
}
