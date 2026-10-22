package app

import (
	"testing"

	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// TestSettingsSidebarAttention marks each section with something waiting for
// the user with a dot beside its sidebar item, and clears it once handled.
func TestSettingsSidebarAttention(t *testing.T) {
	for _, dark := range []bool{false, true} {
		state, _, store := fixture()
		state.Preferences.UILanguage = runtime.LanguageChinese
		messages := store.Copy()
		page := settings.NewPage()
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		tt.SetDark(dark)
		settle(tt)
		accent := theme.Resolve(dark).Accent
		for _, label := range []copy.Key{copy.NavGeneral, copy.NavDictation, copy.NavModel, copy.NavAbout} {
			item, _ := tt.Find(messages.Text(label))
			if badgeBeside(tt, item, accent) {
				t.Fatalf("handled state shows a dot beside %s", label)
			}
		}

		state.Permissions.Microphone = runtime.ConsentDenied
		state.Permissions.Accessibility = runtime.ConsentUndetermined
		state.Hold = runtime.Hold{Failure: runtime.CodePermissionDenied}
		state.Model.State = runtime.ModelFailed
		state.Update = runtime.Update{Phase: runtime.UpdateAvailable, Version: "0.2.0"}
		state.Account = runtime.Account{Issuer: state.Account.Issuer, Portal: state.Account.Portal, Phase: runtime.AccountSignedIn, Subject: "sub-1", Name: "Ada Lovelace", Email: "ada@example.com"}
		page.Section = settings.Account
		settle(tt)
		for _, label := range []copy.Key{copy.NavGeneral, copy.NavDictation, copy.NavModel, copy.NavAbout} {
			item, ok := tt.Find(messages.Text(label))
			if !ok {
				t.Fatalf("sidebar item %s missing", label)
			}
			if !badgeBeside(tt, item, accent) {
				t.Errorf("%s: no dot to the right of %+v", label, item)
			}
		}
		for _, clear := range []copy.Key{copy.NavOverview, copy.NavMicrophone, copy.NavVocabulary} {
			item, _ := tt.Find(messages.Text(clear))
			if badgeBeside(tt, item, accent) {
				t.Errorf("%s shows a dot", clear)
			}
		}
		name := "settings-sidebar-attention"
		if dark {
			name += "-dark"
		}
		snapshot(t, tt, name)
		// The dot stays visible on the chosen item's fill.
		page.Section = settings.General
		settle(tt)
		snapshot(t, tt, name+"-chosen")
	}
}

// badgeBeside reports the dot's accent fill on item's line, between its
// label and the sidebar's edge.
func badgeBeside(tt *ui.Tester, item ui.Rect, accent ui.Color) bool {
	img := tt.Image()
	scale := float32(img.Bounds().Dx()) / SettingsWidth
	y := int((item.Y + item.H/2) * scale)
	hits := 0
	for x := int((item.X + item.W) * scale); x < int(300*scale); x++ {
		c := img.RGBAAt(x, y)
		if near(c.R, accent.R) && near(c.G, accent.G) && near(c.B, accent.B) {
			hits++
		}
	}
	return hits >= 2
}

func near(a, b uint8) bool { d := int(a) - int(b); return d > -7 && d < 7 }
