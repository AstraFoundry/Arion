package app

import (
	"testing"
	"time"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

// TestAboutContributors opens every contributor from About's stack:
// hovering an avatar names its GitHub account, a click opens its profile,
// and back returns to About.
func TestAboutContributors(t *testing.T) {
	if len(assets.Contributors) == 0 {
		t.Fatal("no embedded contributors")
	}
	for _, dark := range []bool{false, true} {
		state, _, store := fixture()
		state.Preferences.UILanguage = runtime.LanguageChinese
		messages := store.Copy()
		page := settings.NewPage()
		page.Section = settings.About
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		tt.SetDark(dark)
		settle(tt)
		reveal(t, tt, messages.Text(copy.AboutContributors))
		// The row is the card's last; scroll to the end so all of it shows.
		tt.Scroll(SettingsWidth*0.7, SettingsHeight/2, 0, 400)
		settle(tt)
		suffix := ""
		if dark {
			suffix = "-dark"
		}
		snapshot(t, tt, "about-contributors-row"+suffix)

		if err := tt.Click(messages.Text(copy.AboutContributors)); err != nil {
			t.Fatal(err)
		}
		rest(tt, 60*time.Millisecond)
		snapshot(t, tt, "about-contributors-entering"+suffix)
		settle(tt)
		first := assets.Contributors[0]
		box, ok := tt.Find(first.Login)
		if !ok {
			t.Fatalf("contributor %s missing: %q", first.Login, tt.Texts())
		}
		tt.Move(box.X+box.W/2, box.Y+box.H/2)
		rest(tt, 1500*time.Millisecond)
		if !hasText(tt, first.Login) {
			t.Errorf("no tooltip naming %s: %q", first.Login, tt.Texts())
		}
		snapshot(t, tt, "about-contributors-page"+suffix)
		if err := tt.Click(first.Login); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if got := tt.OpenedURLs(); len(got) != 1 || got[0] != first.Profile {
			t.Errorf("opened %q, want %s", got, first.Profile)
		}
		if err := tt.Click(messages.Text(copy.Back)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if !hasText(tt, messages.Text(copy.AboutLicense)) {
			t.Errorf("back did not return to About: %q", tt.Texts())
		}
	}
}
