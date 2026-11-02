package signin

import (
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Choice belongs to this onboarding window, never to a later account or refresh.
type Choice struct {
	TermsAccepted bool
	ShareUsage    bool
}

// NewChoice keeps optional usage sharing off until the user selects it.
func NewChoice() Choice { return Choice{TermsAccepted: true} }

// Onboarding keeps both choices beside the documents below browser sign-in.
func Onboarding(c *ui.Context, store runtime.Store, tokens theme.Tokens, choice *Choice) {
	messages := store.Copy()
	if store.State.Account.Portal == "" {
		start(c, store, tokens, kit.Outline, messages.Text(copy.AccSignIn), choice.TermsAccepted, choice.ShareUsage)
	} else {
		methods(c, store, tokens, choice.TermsAccepted, choice.ShareUsage)
	}
	origin := config.SiteURL
	if store.State.Preferences.UILanguage == runtime.LanguageChinese {
		origin += "/zh"
	}
	ui.Column(c).Gap(6).Children(func() {
		ui.Checkbox(c.Key("accept-terms"), &choice.TermsAccepted, messages.Text(copy.AccTermsAccept)).FontSize(theme.Caption)
		ui.Row(c).Padding(0, 0, 0, 24).Gap(12).Children(func() {
			ui.Link(c, messages.Text(copy.AccTerms), origin+config.TermsPath).FontSize(theme.Caption).TextColor(tokens.Text)
			ui.Link(c, messages.Text(copy.AccPrivacy), origin+config.PrivacyPath).FontSize(theme.Caption).TextColor(tokens.Text)
		})
		ui.Checkbox(c.Key("share-usage"), &choice.ShareUsage, messages.Text(copy.AccUsageAgree)).FontSize(theme.Caption)
		ui.Link(c, messages.Text(copy.AccUsageData), origin+config.UsageDataPath).Margin(0, 0, 0, 24).FontSize(theme.Caption).TextColor(tokens.Text)
	})
}
