// Package usagesharing is the account's choice to share daily usage totals.
// The onboarding sign-in page offers it as a box beside the sign-in buttons;
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

// Box is the sign-in page's "Share usage data" check box, unchecked until the
// user checks it. Checking it answers nothing until the sign-in succeeds.
func Box(c *ui.Context, store runtime.Store, tokens theme.Tokens, share *bool) {
	messages := store.Copy()
	ui.Column(c).Gap(2).Children(func() {
		ui.Checkbox(c.Key("share-usage"), share, messages.Text(copy.AccUsageShare)).Disabled(store.State.Account.Busy())
		ui.Text(c, messages.Text(copy.AccUsageShareDetail)).FontSize(theme.Caption).TextColor(tokens.Muted).Margin(0, 0, 0, 24)
	})
}
