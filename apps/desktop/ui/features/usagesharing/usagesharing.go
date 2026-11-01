// Package usagesharing is the account's choice to share daily usage totals.
// Settings › Account changes it with a switch.
package usagesharing

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Switch is the settings row that shares usage data, shown while signed in.
// It follows the platform's record and waits while the choice is read or
// saved.
func Switch(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	a := store.State.Account
	if !a.SignedIn() {
		return
	}
	messages := store.Copy()
	shared := a.UsageSharing == runtime.UsageShared
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.AccUsageShare), messages.Text(copy.AccUsageShareDetail), func() {
			if ui.Switch(c.Key("usage-sharing"), &shared).Label(messages.Text(copy.AccUsageShare)).Disabled(a.UsageLoading || a.UsageSaving).Changed() {
				store.Actions.SetUsageSharing(shared)
			}
		})
	})
}
