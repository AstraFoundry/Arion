package app

import (
	platform "runtime"
	"testing"
	"time"

	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/pages/overlay"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestNativeOptionAltLegendsAcrossPanelAndDictation(t *testing.T) {
	modifier := "Alt"
	if platform.GOOS == "darwin" {
		modifier = "Option"
	}
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, _, store := fixture()
		state.Preferences.UILanguage = locale
		state.Preferences.HoldKey, state.Preferences.TranslateKey = "alt+i", "alt+k"
		panel := &menubar.Panel{}
		tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight)
		if !hasText(tt, modifier+"+I") || hasText(tt, "ALT+I") {
			t.Fatalf("%s panel uses an incorrect modifier legend: %q", locale, tt.Texts())
		}
		snapshot(t, tt, "shortcut-native-panel-"+string(locale))
		state.Dictation = runtime.Dictation{SessionID: "native-legend", State: runtime.StateCapturing, Levels: levels(40), Changed: time.Now()}
		pill := overlay.NewPage()
		tt = ui.NewTester(func(c *ui.Context) { pill.Render(c, store) }, OverlayWidth, OverlayHeight)
		if !hasText(tt, modifier+"+I") || hasText(tt, "ALT+I") {
			t.Fatalf("%s dictation pill uses an incorrect modifier legend: %q", locale, tt.Texts())
		}
		snapshot(t, tt, "shortcut-native-pill-"+string(locale))
		state.Dictation.Translating = true
		tt.Frame()
		if !hasText(tt, modifier+"+K") || hasText(tt, "ALT+K") {
			t.Fatalf("%s translation pill uses an incorrect modifier legend: %q", locale, tt.Texts())
		}
	}
}
