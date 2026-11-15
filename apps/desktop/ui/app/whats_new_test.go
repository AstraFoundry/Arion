package app

import (
	"slices"
	"testing"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

// TestWhatsNewOverSettings shows the release highlights over Settings,
// skips them, and opens them again from About.
func TestWhatsNewOverSettings(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, actions, store := fixture()
		state.Version = "0.1.6"
		state.Preferences.UILanguage = locale
		actions.state = state
		state.WhatsNew = runtime.WhatsNew{Seq: 1, Slides: []runtime.WhatsNewSlide{
			{ID: "selection-actions", Version: "0.1.6"}, {ID: "selection-vocabulary", Version: "0.1.6"}}}
		messages := store.Copy()
		page := settings.NewPage()
		page.Section = settings.About
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		settle(tt)
		if !slices.Contains(tt.Texts(), messages.Text(copy.WhatsNewSelectionTitle)) {
			t.Fatalf("%s: highlights missing over Settings: %q", locale, tt.Texts())
		}
		snapshot(t, tt, "whats-new-settings-"+string(locale))
		if err := tt.Click(messages.Text(copy.Skip)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if !actions.called("whats-new-dismiss") || slices.Contains(tt.Texts(), messages.Text(copy.WhatsNewSelectionTitle)) {
			t.Fatalf("%s: Skip dismisses: %v %q", locale, actions.calls, tt.Texts())
		}
		if err := tt.Click(messages.Text(copy.AboutWhatsNew)); err != nil {
			t.Fatal(err)
		}
		if !actions.called("whats-new") {
			t.Fatalf("%s: About reopens the highlights: %v", locale, actions.calls)
		}
	}
}

// TestAboutHidesWhatsNewWithoutHighlights keeps the entry away from a
// version no release with highlights precedes.
func TestAboutHidesWhatsNewWithoutHighlights(t *testing.T) {
	state, _, store := fixture()
	state.Version = "0.1.5"
	page := settings.NewPage()
	page.Section = settings.About
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	if slices.Contains(tt.Texts(), store.Copy().Text(copy.AboutWhatsNew)) {
		t.Fatalf("0.1.5 offers What's new: %q", tt.Texts())
	}
}
