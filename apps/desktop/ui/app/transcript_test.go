package app

import (
	"strings"
	"testing"
	"time"

	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestOriginalTranscriptPanel(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			state.Preferences.Polish, state.Preferences.MatchApp = true, true
			state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
			state.SceneTarget = runtime.SceneTarget{Known: true, AppName: "ChatGPT", Hints: runtime.SceneHints{App: "com.openai.chat", Field: runtime.FieldText}}
			state.LastText = "Can you build and verify locally?"
			state.LastOriginal = "Um, can you build and, uh, verify locally?"
			panel := &menubar.Panel{}
			tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight)
			settle(tt)
			snapshot(t, tt, "panel-original-collapsed-"+string(locale))
			messages := store.Copy()
			result, _ := tt.Find(state.LastText)
			for _, label := range []string{messages.Text(copy.PanelLastDictation), messages.Text(copy.ViewOriginalTranscript)} {
				bounds, ok := tt.Find(label)
				if !ok || bounds.X != result.X {
					t.Fatalf("%q is not aligned with the result: %+v result %+v", label, bounds, result)
				}
			}
			if err := tt.Click(messages.Text(copy.ViewOriginalTranscript)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			tt.Move(0, 0)
			settle(tt)
			rest(tt, 250*time.Millisecond)
			snapshot(t, tt, "panel-original-expanded-"+string(locale))
			tt.SetDark(true)
			settle(tt)
			snapshot(t, tt, "panel-original-expanded-dark-"+string(locale))
			tt.SetDark(false)
			settle(tt)
			for _, label := range []string{messages.Text(copy.HideOriginalTranscript), state.LastOriginal} {
				bounds, ok := tt.Find(label)
				if !ok || bounds.X != result.X {
					t.Fatalf("%q is not aligned with the result: %+v result %+v", label, bounds, result)
				}
			}
			for _, label := range []string{messages.Text(copy.HideOriginalTranscript), messages.Text(copy.CopyOriginal), messages.Text(copy.SettingsMenu), messages.Text(copy.Quit)} {
				bounds, ok := tt.Find(label)
				if !ok || bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > PanelWidth || bounds.Y+bounds.H > PanelHeight {
					t.Fatalf("%q outside panel: %+v", label, bounds)
				}
			}
			state.Preferences.UILanguage = runtime.LanguageChinese
			if locale == runtime.LanguageChinese {
				state.Preferences.UILanguage = runtime.LanguageEnglish
			}
			settle(tt)
			if !hasText(tt, store.Copy().Text(copy.HideOriginalTranscript)) || !hasText(tt, state.LastOriginal) {
				t.Fatalf("language switch lost preview: %q", tt.Texts())
			}
			state.Preferences.UILanguage = locale
			state.LastOriginal = strings.Repeat("Um, we should verify the build locally before shipping. ", 24)
			settle(tt)
			if err := tt.Click(messages.Text(copy.ViewOriginalTranscript)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			tt.Move(0, 0)
			rest(tt, 250*time.Millisecond)
			snapshot(t, tt, "panel-original-long-"+string(locale))
			if err := tt.Click(messages.Text(copy.CopyOriginal)); err != nil {
				t.Fatal(err)
			}
			if !actions.called("copy:" + state.LastOriginal) {
				t.Fatal("long original was truncated when copied")
			}
			sourceBefore, _ := tt.Find(state.LastOriginal)
			tt.Scroll(PanelWidth/2, sourceBefore.Y+8, 0, 500)
			settle(tt)
			sourceAfter, _ := tt.Find(state.LastOriginal)
			if sourceAfter.Y >= sourceBefore.Y {
				t.Fatalf("long source did not scroll: before %+v after %+v", sourceBefore, sourceAfter)
			}
			state.LastText = "The next dictation."
			settle(tt)
			bounds, ok := tt.Find(state.LastText)
			if !ok || bounds.Y < 0 || bounds.Y+bounds.H > PanelHeight || !hasText(tt, messages.Text(copy.ViewOriginalTranscript)) {
				t.Fatalf("new result retained scroll or preview: %+v texts %q", bounds, tt.Texts())
			}
			if err := tt.Click(messages.Text(copy.SettingsMenu)); err != nil {
				t.Fatal(err)
			}
			if !actions.called("settings") {
				t.Fatal("long original displaced settings")
			}
		})
	}
}
