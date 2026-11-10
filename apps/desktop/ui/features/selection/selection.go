// Package selection turns the selection popover and the assistants it may
// summon on or off. Both stay off until the user enables them.
package selection

import (
	"slices"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

type client struct {
	id            string
	title, detail copy.Key
}

// clients follow the backend's default order: Claude precedes ChatGPT.
var clients = []client{
	{runtime.SelectionClientClaude, copy.SelectionClaude, copy.SelectionClaudeDetail},
	{runtime.SelectionClientChatGPT, copy.SelectionChatGPT, copy.SelectionChatGPTDetail},
}

// ClientEnabled reports whether the stored preferences allow client id.
func ClientEnabled(prefs runtime.Preferences, id string) bool {
	for _, c := range prefs.Selection.Clients {
		if c.ID == id {
			return c.Enabled
		}
	}
	return false
}

// Settings is the master switch with the client rows disclosed beneath it.
// It writes Preferences.Selection through UpdatePreferences only.
func Settings(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	prefs := store.State.Preferences
	on := prefs.Selection.Enabled
	kit.Row(c, tokens, messages.Text(copy.SelectionEnabled), messages.Text(copy.SelectionEnabledDetail), func() {
		if ui.Switch(c.Key("selection-enabled"), &on).Label(messages.Text(copy.SelectionEnabled)).Changed() {
			store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Selection.Enabled = on })
		}
	})
	kit.Disclosure(c.Key("selection-clients"), on, func() {
		for _, item := range clients {
			kit.Divider(c, tokens)
			enabled := ClientEnabled(prefs, item.id)
			kit.Row(c, tokens, messages.Text(item.title), messages.Text(item.detail), func() {
				if ui.Switch(c.Key("selection-client-"+item.id), &enabled).Label(messages.Text(item.title)).Changed() {
					id, value := item.id, enabled
					store.Actions.UpdatePreferences(func(p *runtime.Preferences) { setClient(p, id, value) })
				}
			})
		}
	})
}

// setClient copies the client list so the previous preferences value is
// not mutated through its shared backing array. Normalize keeps both
// built-in clients in that list.
func setClient(p *runtime.Preferences, id string, enabled bool) {
	list := slices.Clone(p.Selection.Clients)
	for i := range list {
		if list[i].ID == id {
			list[i].Enabled = enabled
		}
	}
	p.Selection.Clients = list
}
