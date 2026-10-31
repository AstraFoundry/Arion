// Package signin is the AstraFoundry account for hosted sync: signing in
// through the browser and signing out. Dictation, memory and learning never
// depend on it.
package signin

import (
	"bytes"
	"strings"

	"arion.local/desktop/ui/entities/account"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Start is the button that opens the browser sign-in.
func Start(c *ui.Context, store runtime.Store, tokens theme.Tokens, variant kit.Variant, label string) {
	if store.State.Account.Disabled {
		return
	}
	if kit.Button(c.Key("sign-in"), tokens, variant, label, !store.State.Account.Busy()).Clicked() {
		store.Actions.SignIn()
	}
}

// Methods are the ways to sign in or create an AstraFoundry account. Each
// opens the same browser sign-in, where the issuer's page offers them all;
// the button only says which one the user means to pick there.
func Methods(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if store.State.Account.Disabled {
		return
	}
	messages := store.Copy()
	enabled := !store.State.Account.Busy()
	ui.Column(c).Gap(8).AlignItems(ui.Stretch).Children(func() {
		for _, method := range []struct {
			key   string
			icon  *ui.SVG
			label copy.Key
		}{{"google", icons.Google, copy.AccContinueGoogle}, {"github", icons.GitHub, copy.AccContinueGitHub}, {"email", icons.Mail, copy.AccContinueEmail}} {
			button := ui.ButtonBase(c.Key("sign-in-"+method.key)).Height(36).Padding(0, 14).Gap(10).Radius(theme.RadiusControl).
				Justify(ui.Center).AlignItems(ui.Center).Disabled(!enabled)
			kit.Style(button, tokens, kit.Outline, enabled)
			button.Children(func() {
				// An image keeps the Google mark's own colors; an icon would tint it.
				ui.Image(c, method.icon).Size(15, 15)
				ui.Text(c, messages.Text(method.label)).FontSize(theme.Body).FontWeight(500).SingleLine()
			})
			if button.Clicked() {
				store.Actions.SignIn()
			}
		}
	})
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
	if store.State.Account.Disabled {
		return
	}
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
			title := a.Issuer
			if title == "" {
				title = messages.Text(copy.AccTitle)
			}
			kit.Row(c, tokens, title, "", func() {
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

				signOut(c, store, tokens)
			})
		case account.Profile:
			profile(c, store, tokens)
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
	ui.Row(c).Gap(12).Padding(14, 16).AlignItems(ui.Center).Children(func() {
		Avatar(c, tokens, a, 40)
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(theme.Heading).FontWeight(600).ContentTransition(ui.ContentTransition{})
			if detail != "" {
				ui.Text(c, detail).FontSize(theme.Caption).TextColor(tokens.Muted).Selectable()
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if a.Busy() {
				ui.Spinner(c).Size(14, 14)
			}
			signOut(c, store, tokens)
		})
	})
}

// Avatar is the profile picture, or the initials of the name on a color
// of its own, or the person glyph without either. A picture that does not
// decode shows the initials.
func Avatar(c *ui.Context, tokens theme.Tokens, a runtime.Account, size float32) ui.Element {
	name := strings.TrimSpace(a.Name)
	if name == "" {
		name = strings.TrimSpace(a.Email)
	}
	bitmap := picture(a.PictureData)
	if name == "" && bitmap == nil {
		return glyph(c, tokens, size)
	}
	return ui.Avatar(c, name, bitmap).FontSize(size / 2.25)
}

func glyph(c *ui.Context, tokens theme.Tokens, size float32) ui.Element {
	disc := ui.Box(c).Size(size, size).Radius(size / 2).Background(tokens.Canvas).Justify(ui.Center).AlignItems(ui.Center)
	disc.Children(func() { ui.Icon(c, icons.CircleUser).FontSize(size * 0.62).TextColor(tokens.Muted) })
	return disc
}

// pictures keeps the last decoded picture: decoding is not free.
var pictures struct {
	data   []byte
	bitmap *ui.Bitmap
}

func picture(data []byte) *ui.Bitmap {
	if len(data) == 0 {
		return nil
	}
	if !bytes.Equal(pictures.data, data) {
		pictures.data = data
		pictures.bitmap, _ = ui.DecodeBitmap(data)
	}
	return pictures.bitmap
}

// SidebarCard is the account at the foot of the settings sidebar, as macOS
// System Settings shows it: the picture and name with the email, or the
// account section while signed out. It reports a click; selected marks its section showing.
func SidebarCard(c *ui.Context, store runtime.Store, tokens theme.Tokens, selected bool) bool {
	messages := store.Copy()
	a := store.State.Account
	card := ui.ButtonBase(c.Key("account-card")).Gap(10).Padding(8, 10).Radius(theme.RadiusCard).AlignItems(ui.Center).
		Border(1, tokens.CardBorder)
	// Selection and hover use the sidebar's neutral fills, not the accent.
	surface := c.Theme()
	switch {
	case selected:
		card.Background(surface.SurfacePressed)
	case card.Hovered():
		card.Background(surface.SurfaceHover)
	default:
		card.Background(tokens.Card)
	}
	card.Children(func() {
		if !a.SignedIn() {
			glyph(c, tokens, 32)
			ui.Text(c, messages.Text(copy.NavAccount)).FontSize(theme.Body).FontWeight(600).SingleLine().Grow(1)
			return
		}
		title, detail, key := account.Identity(a)
		if key != "" {
			title = messages.Text(key)
		}
		Avatar(c, tokens, a, 32)
		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			ui.Text(c, title).FontSize(theme.Body).FontWeight(600).SingleLine()
			if detail != "" {
				ui.Text(c, detail).FontSize(theme.Caption).TextColor(tokens.Muted).SingleLine()
			}
		})
	})
	return card.Clicked()
}

// Portal links the signed-in account's pages on the AstraFoundry account
// portal, opened in the browser. A custom issuer has none.
func Portal(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	a := store.State.Account
	if a.Portal == "" || !a.SignedIn() {
		return
	}
	messages := store.Copy()
	kit.Group(c, tokens, messages.Text(copy.AccPortalGroup), func() {
		kit.Card(c, tokens).Children(func() {
			for i, page := range account.PortalPages {
				if i > 0 {
					kit.Divider(c, tokens)
				}
				url := a.Portal + page.Path
				row := kit.RowFrame(ui.ButtonBase(c.Key(page.Path))).Gap(12).AlignItems(ui.Center).Label(messages.Text(page.Title))
				if row.Hovered() {
					row.Background(tokens.AccentSoft.Alpha(0.5).Over(tokens.Card))
				}
				row.Children(func() {
					ui.Text(c, messages.Text(page.Title)).FontSize(theme.Body).FontWeight(500).Grow(1)
					ui.Icon(c, icons.ExternalLink).FontSize(13).TextColor(tokens.Muted)
				})
				if row.Clicked() {
					c.OpenURL(url)
				}
			}
		})
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
	kit.Notice(c.Key("account-notice"), tokens, tokens.Danger, store.Copy().Text(key), nil).
		Padding(0, 4).Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}})
}

// HostedSync reports that the signed-in account's access token carries the
// hosted-sync entitlement. Without that entitlement this card is omitted.
// Setting up this device starts here once the desktop sync client exists.
func HostedSync(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	a := store.State.Account
	if a.Disabled || !a.HostedSync {
		return
	}
	messages := store.Copy()
	kit.Card(c, tokens).Children(func() {
		line(c, tokens, icons.CircleCheck, tokens.Success, messages.Text(copy.AccSyncAvailable), messages.Text(copy.AccSyncDeviceLater), nil)
	})
}

// UsageReporting says whether this session reports consented daily usage.
// The choice lives on the portal's data page; a new choice reaches Arion
// with the next token, which checking again fetches.
func UsageReporting(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	a := store.State.Account
	manage := func() {
		if a.Portal != "" && kit.Button(c.Key("usage-manage"), tokens, kit.Outline, messages.Text(copy.AccUsageManage), true).Clicked() {
			c.OpenURL(a.Portal + "/data")
		}
	}
	kit.Card(c, tokens).Children(func() {
		if a.UsageReporting {
			line(c, tokens, icons.CircleCheck, tokens.Success, messages.Text(copy.AccUsageOn), messages.Text(copy.AccUsageOnDetail), manage)
			return
		}
		line(c, tokens, icons.Info, tokens.Muted, messages.Text(copy.AccUsageOff), messages.Text(copy.AccUsageOffDetail), func() {
			manage()
			if kit.Button(c.Key("usage-check"), tokens, kit.Outline, messages.Text(copy.AccCheckAgain), !a.Busy()).Clicked() {
				store.Actions.RefreshAccount()
			}
		})
	})
}

// line is a settings row led by an icon that carries the state's color.
func line(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, color ui.Color, title, detail string, control func()) {
	kit.RowFrame(ui.Row(c)).Gap(12).AlignItems(ui.Center).Children(func() {
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

// Terms presents the product terms before opening browser authentication.
func Terms(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	origin := config.SiteURL
	if store.State.Preferences.UILanguage == runtime.LanguageChinese {
		origin += "/zh"
	}
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, messages.Text(copy.AccTermsNotice)).FontSize(theme.Caption).TextColor(tokens.Muted)
		ui.Row(c).Gap(12).Children(func() {
			ui.Link(c, messages.Text(copy.AccTerms), origin+config.TermsPath).FontSize(theme.Caption).TextColor(tokens.Text)
			ui.Link(c, messages.Text(copy.AccPrivacy), origin+config.PrivacyPath).FontSize(theme.Caption).TextColor(tokens.Text)
		})
	})
}
