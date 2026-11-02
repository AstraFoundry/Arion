package signin

import (
	"strings"

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
	ui.Column(c).Gap(8).Children(func() {
		consentRow(c.Key("accept-terms"), tokens, &choice.TermsAccepted, messages.Text(copy.AccTermsAccept), []consentLink{
			{messages.Text(copy.AccTerms), origin + config.TermsPath},
			{messages.Text(copy.AccPrivacy), origin + config.PrivacyPath},
		})
		consentRow(c.Key("share-usage"), tokens, &choice.ShareUsage, messages.Text(copy.AccUsageAgree), []consentLink{
			{messages.Text(copy.AccUsageData), origin + config.UsageDataPath},
		})
	})
}

type consentLink struct{ label, url string }

// Links retain their own hit areas; ordinary label text toggles the checkbox.
func consentRow(c *ui.Context, tokens theme.Tokens, checked *bool, label string, links []consentLink) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Checkbox(c, checked, "").Label(label)
		ui.RichText(c).FontSize(theme.Caption).TextColor(tokens.Text).NoWrap().Label(label).
			OnClick(func() { *checked = !*checked }).Children(func() {
			remaining := label
			for _, link := range links {
				before, after, found := strings.Cut(remaining, link.label)
				if !found {
					panic("consent link is missing from its localized label")
				}
				ui.Text(c, before)
				ui.Link(c, link.label, link.url).TextColor(tokens.Text).Underline()
				remaining = after
			}
			ui.Text(c, remaining)
		})
	})
}
