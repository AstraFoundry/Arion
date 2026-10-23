package app

import (
	"testing"

	"arion.local/desktop/ui/features/polish"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The compact header grows only after download confirmation. Its panel
// can reverse mid-flight, and reduced motion takes the final layout at once.
func TestLocalPolishDisclosure(t *testing.T) {
	for _, reduce := range []bool{false, true} {
		name := "animated"
		if reduce {
			name = "reduced-motion"
		}
		t.Run(name, func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = runtime.LanguageChinese
			state.Polish = runtime.Polish{State: runtime.PolishOff, SizeBytes: 1280835840}
			tt := ui.NewTester(func(c *ui.Context) {
				tokens := theme.Apply(c)
				ui.Column(c).Width(420).Children(func() { polish.Choice(c, store, tokens) })
			}, 420, 360)
			tt.SetPreferences(ui.Preferences{ReduceMotion: reduce})
			messages := store.Copy()
			cloudY := func() float32 {
				t.Helper()
				r, ok := tt.Find(messages.Text(copy.PolishEngineCloud))
				if !ok {
					t.Fatal("cloud option missing")
				}
				return r.Y
			}
			compact := cloudY()
			if err := tt.Click(messages.Text(copy.PolishEngineLocal)); err != nil {
				t.Fatal(err)
			}
			if cloudY() != compact || actions.called("download-polish") {
				t.Fatal("selection expanded the panel or started a download")
			}
			if err := tt.Click(messages.Text(copy.PolishDownload)); err != nil || !actions.called("download-polish") {
				t.Fatalf("download action: %v", err)
			}
			state.Polish.State, state.Polish.Phase = runtime.PolishProvisioning, runtime.PhaseDownloading
			state.Polish.Done, state.Polish.Total = 4<<20, state.Polish.SizeBytes
			tt.Move(0, 0)
			opening := cloudY()
			snapshot(t, tt, "polish-reveal-opening-"+name)
			if hasText(tt, messages.Text(copy.PolishReady)) {
				t.Fatal("downloading model shows a ready check")
			}
			// Reverse before the opening animation has settled.
			state.Preferences.PolishEngine = runtime.PolishCloud
			tt.Frame()
			rest(tt, theme.Disclosure.Duration)
			tt.Frame()
			if cloudY() != compact || hasText(tt, messages.Text(copy.PolishCancelDownload)) {
				t.Fatal("interrupted panel did not close completely")
			}
			state.Preferences.PolishEngine = runtime.PolishLocal
			tt.Frame()
			snapshot(t, tt, "polish-reveal-reopening-"+name)
			rest(tt, theme.Disclosure.Duration)
			tt.Frame()
			expanded := cloudY()
			if expanded <= compact || !reduce && opening >= expanded || reduce && opening != expanded {
				t.Fatalf("disclosure geometry: compact %v, opening %v, expanded %v", compact, opening, expanded)
			}
			snapshot(t, tt, "polish-reveal-expanded-"+name)
			state.Polish.State, state.Polish.Installed = runtime.PolishLoading, true
			tt.Frame()
			if hasText(tt, messages.Text(copy.PolishReady)) {
				t.Fatal("installed but loading model shows a ready check")
			}
			state.Polish.State = runtime.PolishReady
			tt.Frame()
			snapshot(t, tt, "polish-reveal-closing-"+name)
			rest(tt, theme.Disclosure.Duration)
			tt.Frame()
			if cloudY() != compact || !hasText(tt, messages.Text(copy.PolishReady)) || hasText(tt, messages.Text(copy.PolishCancelDownload)) {
				t.Fatal("ready model did not return to a compact header with a check")
			}
			snapshot(t, tt, "polish-reveal-ready-"+name)
		})
	}
}
