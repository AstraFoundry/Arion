// Package signin is the AstraFoundry account for hosted sync: signing in
// through the browser, signing out, and joining the hosted-sync waitlist.
// Dictation, memory and learning never depend on it.
package signin

import (
	"arion.local/desktop/ui/entities/account"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Start is the button that opens the browser sign-in.
func Start(c *ui.Context, store runtime.Store, tokens theme.Tokens, variant kit.Variant, label string) {
	if kit.Button(c.Key("sign-in"), tokens, variant, label, !store.State.Account.Busy()).Clicked() {
		store.Actions.SignIn()
	}
}

// Check asks once, while an account view shows, whether a refresh token is
// stored. The read is local and off the main thread.
func Check(store runtime.Store) {
	if account.NeedsCredentialCheck(store.State.Account) {
		store.Actions.CheckStoredAccount()
	}
}

// Card is the account in its current stage, followed by the last failure.
func Card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	ui.Column(c).Gap(theme.GapRelated).Children(func() {
		card(c, store, tokens)
		Notice(c, store, tokens)
	})
}

func card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	a := store.State.Account
	kit.Card(c, tokens).Children(func() {
		switch account.StageOf(a) {
		case account.SignedOut:
			kit.Row(c, tokens, messages.Text(copy.AccTitle), messages.Text(copy.AccNoDecrypt), func() {
				Start(c, store, tokens, kit.Outline, messages.Text(copy.AccSignIn))
			})
		case account.Waiting:
			kit.Row(c, tokens, messages.Text(copy.AccTitle), messages.Text(copy.AccWaiting), func() {
				ui.Spinner(c).Size(14, 14)
				if kit.Button(c.Key("cancel-sign-in"), tokens, kit.Outline, messages.Text(copy.AccCancel), true).Clicked() {
					store.Actions.CancelSignIn()
				}
			})
		case account.Checking:
			kit.Row(c, tokens, messages.Text(copy.AccTitle), messages.Text(copy.AccChecking), func() { ui.Spinner(c).Size(14, 14) })
		case account.Stored:
			kit.Row(c, tokens, messages.Text(copy.AccTitle), messages.Text(copy.AccStoredDetail), func() {
				if a.Busy() {
					ui.Spinner(c).Size(14, 14)
				}
				if kit.Button(c.Key("reconnect"), tokens, kit.Outline, messages.Text(copy.AccReconnect), !a.Busy()).Clicked() {
					store.Actions.RefreshAccount()
				}
				signOut(c, store, tokens)
			})
			kit.Divider(c, tokens)
			ui.Text(c, messages.Text(copy.AccSignOutKeeps)).FontSize(theme.Caption).TextColor(tokens.Muted).Padding(10, 16)
		case account.Profile:
			profile(c, store, tokens)
			kit.Divider(c, tokens)
			ui.Text(c, messages.Text(copy.AccSignOutKeeps)).FontSize(theme.Caption).TextColor(tokens.Muted).Padding(10, 16)
		}
	})
}

func profile(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	a := store.State.Account
	title, detail, key := account.Identity(a)
	if key != "" {
		title = messages.Text(key)
	}
	line(c, tokens, icons.CircleUser, tokens.Muted, title, detail, func() {
		if a.Busy() {
			ui.Spinner(c).Size(14, 14)
		}
		signOut(c, store, tokens)
	})
}

// signOut revokes and deletes the stored refresh token, with or without a
// session in memory.
func signOut(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	a := store.State.Account
	if kit.Button(c.Key("sign-out"), tokens, kit.Outline, store.Copy().Text(copy.AccSignOut), !a.Busy()).Clicked() {
		store.Actions.SignOut()
	}
}

// Notice states the last account failure, or a sign-out whose revocation
// AstraFoundry did not confirm. Nothing shows otherwise.
func Notice(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	key := account.Notice(store.State.Account)
	if key == "" {
		return
	}
	ui.Row(c.Key("account-notice")).Gap(6).Padding(0, 4).Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}}).Children(func() {
		ui.Icon(c, icons.TriangleAlert).FontSize(13).TextColor(tokens.Danger).Margin(1, 0, 0, 0)
		ui.Text(c, store.Copy().Text(key)).FontSize(theme.Caption).Shrink(1)
	})
}

// HostedSync is the signed-in account's hosted-sync access: available, or
// the waitlist with its phase.
func HostedSync(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	a := store.State.Account
	kit.Card(c, tokens).Children(func() {
		if a.HostedSync {
			// Device enrollment (SYNC-02) starts here once the desktop sync
			// client exists.
			line(c, tokens, icons.CircleCheck, tokens.Success, messages.Text(copy.AccSyncAvailable), messages.Text(copy.AccSyncDeviceLater), nil)
			return
		}
		waitlist(c, store, tokens)
	})
}

func waitlist(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	a := store.State.Account
	button := func(label string, action func()) {
		if kit.Button(c.Key(label), tokens, kit.Outline, label, !a.Busy()).Clicked() {
			action()
		}
	}
	icon, color := icons.Info, tokens.Muted
	var control func()
	switch a.Waitlist.Phase {
	case runtime.AccountWaitlistJoining:
		control = func() { ui.Spinner(c).Size(14, 14) }
	case runtime.AccountWaitlistJoinedPending:
		icon = icons.Clock
		control = func() { button(messages.Text(copy.AccCheckAgain), store.Actions.RefreshAccount) }
	case runtime.AccountWaitlistAlreadyEntitled:
		// The access token predates the entitlement; a refresh carries it.
		icon, color = icons.CircleCheck, tokens.Success
		control = func() { button(messages.Text(copy.AccCheckAgain), store.Actions.RefreshAccount) }
	case runtime.AccountWaitlistClosed:
	case runtime.AccountWaitlistError:
		icon, color = icons.TriangleAlert, tokens.Danger
		control = func() { button(messages.Text(copy.Retry), store.Actions.JoinHostedSyncWaitlist) }
	default:
		control = func() { button(messages.Text(copy.AccWaitlistJoin), store.Actions.JoinHostedSyncWaitlist) }
	}
	line(c, tokens, icon, color, messages.Text(account.WaitlistStatus(a.Waitlist.Phase)), messages.Text(copy.AccSelfHosted), control)
}

// line is a settings row led by an icon that carries the state's color.
func line(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, color ui.Color, title, detail string, control func()) {
	ui.Row(c).Gap(12).Padding(12, 16).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon).FontSize(16).TextColor(color)
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(theme.Body).FontWeight(500).ContentTransition(ui.ContentTransition{})
			if detail != "" {
				ui.Text(c, detail).FontSize(theme.Caption).TextColor(tokens.Muted)
			}
		})
		if control != nil {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(control)
		}
	})
}
