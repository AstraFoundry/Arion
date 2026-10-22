package app

import (
	"testing"
	"time"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

// TestAboutCommunity opens each community chat from its app button in
// About, which names the app on hover.
func TestAboutCommunity(t *testing.T) {
	for _, dark := range []bool{false, true} {
		state, _, store := fixture()
		state.Preferences.UILanguage = runtime.LanguageChinese
		messages := store.Copy()
		page := settings.NewPage()
		page.Section = settings.About
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		tt.SetDark(dark)
		settle(tt)
		reveal(t, tt, messages.Text(copy.AboutCommunity))
		tt.Scroll(SettingsWidth*0.7, SettingsHeight/2, 0, 400)
		settle(tt)
		var want []string
		for _, app := range []struct {
			name copy.Key
			url  string
		}{{copy.AboutDiscord, config.DiscordURL}, {copy.AboutFeishu, config.FeishuURL}} {
			box, ok := tt.Find(messages.Text(app.name))
			if !ok {
				t.Fatalf("no %s button: %q", app.name, tt.Texts())
			}
			tt.Move(box.X+box.W/2, box.Y+box.H/2)
			rest(tt, 1500*time.Millisecond)
			name := "about-community-" + string(app.name)
			if dark {
				name += "-dark"
			}
			snapshot(t, tt, name)
			if err := tt.Click(messages.Text(app.name)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			want = append(want, app.url)
		}
		if got := tt.OpenedURLs(); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("opened %q, want %q", got, want)
		}
	}
}
