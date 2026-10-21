package app

import (
	"testing"
	"time"

	"arion.local/desktop/ui/features/scene"
	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// SCN-A05: the panel shows the scene of the app that was frontmost, says
// whether it is automatic, and switching it keeps the choice for that site.
func TestSceneCard(t *testing.T) {
	state, actions, store := fixture()
	state.Preferences.Polish, state.Preferences.MatchApp = true, true
	state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Padding(16).Children(func() { scene.Card(c, store, theme.Resolve(false)) })
	}, menubar.Width, 320)
	en := copy.For(copy.English)
	settle(tt)
	if hasText(tt, en.Text(copy.SceneChat)) {
		t.Fatal("card shown before a target was read")
	}
	state.SceneTarget = runtime.SceneTarget{Known: true, AppName: "Google Chrome",
		Hints: runtime.SceneHints{App: "com.google.Chrome", Field: runtime.FieldText, Site: "mail.google.com"}}
	settle(tt)
	if !hasText(tt, en.Text(copy.SceneEmail)) || !hasText(tt, "mail.google.com") || !hasText(tt, en.Text(copy.SceneAutomatic)) {
		t.Fatalf("automatic email card: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-scene-automatic")
	if err := tt.Click(en.Text(copy.SceneCardLabel)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	for _, s := range append([]runtime.Scene{""}, runtime.ChoosableScenes...) {
		label := en.Text(copy.SceneAutomatic)
		if s != "" {
			label = en.Text(scene.Label(s))
		}
		if !hasText(tt, label) {
			t.Fatalf("option %q missing: %q", label, tt.Texts())
		}
	}
	snapshot(t, tt, "panel-scene-open")
	if err := tt.Click(en.Text(copy.SceneWork)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.SceneOverrides["mail.google.com"] != runtime.SceneWork || !actions.called("scene:work") {
		t.Fatalf("choice not kept: %v", state.Preferences.SceneOverrides)
	}
	if !hasText(tt, en.Text(copy.SceneWork)) || !hasText(tt, en.Text(copy.SceneChosen)) {
		t.Fatalf("chosen card: %q", tt.Texts())
	}
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	if zh := copy.For(copy.SimplifiedChinese); !hasText(tt, zh.Text(copy.SceneWork)) || !hasText(tt, zh.Text(copy.SceneChosen)) {
		t.Fatalf("not retranslated: %q", tt.Texts())
	}
	rest(tt, 400*time.Millisecond)
	snapshot(t, tt, "panel-scene-chosen-zh-CN")
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	if err := tt.Click(en.Text(copy.SceneCardLabel)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(en.Text(copy.SceneAutomatic)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if len(state.Preferences.SceneOverrides) != 0 || !hasText(tt, en.Text(copy.SceneEmail)) {
		t.Fatalf("automatic not restored: %v", state.Preferences.SceneOverrides)
	}
	// Scenes do not apply without polishing or with Match the app off.
	state.Preferences.MatchApp = false
	settle(tt)
	if hasText(tt, en.Text(copy.SceneEmail)) {
		t.Fatal("card shown with Match the app off")
	}
}

// The panel holds the scene card and four lines of the last dictation.
func TestPanelWithSceneCard(t *testing.T) {
	state, _, store := fixture()
	state.Preferences.Polish, state.Preferences.MatchApp = true, true
	state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
	state.SceneTarget = runtime.SceneTarget{Known: true, AppName: "Cursor", Hints: runtime.SceneHints{App: "com.todesktop.230313mzl4w4u92", Field: runtime.FieldText}}
	state.LastText = "帮我改一下 client.ts，请求失败的时候重试三次，超时设成五秒，然后在 README 里补一段说明，再加一个单元测试覆盖重试逻辑和超时逻辑，最后跑一遍完整测试。"
	panel := &menubar.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight)
	settle(tt)
	en := copy.For(copy.English)
	card, ok := tt.Find(en.Text(copy.ScenePrompt))
	quit, _ := tt.Find(en.Text(copy.Quit))
	text, _ := tt.Find(state.LastText)
	if !ok || text.Y+text.H > quit.Y || card.Y > text.Y {
		t.Fatalf("layout: card %+v text %+v quit %+v", card, text, quit)
	}
	snapshot(t, tt, "panel-scene")
	// The select's popup fits in the panel below the card.
	if err := tt.Click(en.Text(copy.SceneCardLabel)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	search, ok := tt.Find(en.Text(copy.SceneSearch))
	if !ok || search.Y < card.Y || search.Y+search.H > PanelHeight {
		t.Fatalf("popup: search %+v card %+v", search, card)
	}
	snapshot(t, tt, "panel-scene-select")
}
