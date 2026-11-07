package app

import (
	"strings"
	"testing"

	"arion.local/desktop/ui/pages/feedback"
	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestFeedbackWindow(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, actions, store := fixture()
		state.Preferences.UILanguage = locale
		messages := store.Copy()
		page := &feedback.Page{}
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, FeedbackWidth, FeedbackHeight)
		settle(tt)
		prompt := feedback.Prompt(store)
		for _, want := range []string{config.FeedbackSkillURL, state.Version} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("%s: prompt %q lacks %q", locale, prompt, want)
			}
		}
		if !hasText(tt, messages.Text(copy.FeedbackTitle)) || !hasText(tt, prompt) {
			t.Fatalf("%s: %q", locale, tt.Texts())
		}
		snapshot(t, tt, "feedback-"+string(locale))
		// The recording loops; reduced motion holds the opened issue, whose
		// frame has no copy button of its own to confuse the real one.
		tt.SetPreferences(ui.Preferences{ReduceMotion: true})
		settle(tt)
		if !hasText(tt, messages.Text(copy.FeedbackDemoOpened)) {
			t.Fatalf("%s: reduced motion %q", locale, tt.Texts())
		}
		if err := tt.Click(messages.Text(copy.Copy)); err != nil {
			t.Fatal(err)
		}
		// Check within the confirmation, however slow the frames are.
		tt.Frame()
		tt.Frame()
		if !actions.called("copy:"+prompt) || !hasText(tt, messages.Text(copy.Copied)) {
			t.Fatalf("%s: copy %v, texts %q", locale, actions.calls, tt.Texts())
		}
		if err := tt.Click(messages.Text(copy.FeedbackIssueYourself)); err != nil {
			t.Fatal(err)
		}
		if got := tt.OpenedURLs(); len(got) != 1 || got[0] != config.IssuesURL {
			t.Fatalf("%s: opened %q", locale, got)
		}
		snapshot(t, tt, "feedback-still-"+string(locale))
		tt.SetDark(true)
		settle(tt)
		snapshot(t, tt, "feedback-dark-"+string(locale))
	}
}

func TestFeedbackPromptNamesDevelopmentBuilds(t *testing.T) {
	state, _, store := fixture()
	state.Version = ""
	if prompt := feedback.Prompt(store); !strings.Contains(prompt, store.Copy().Text(copy.FeedbackDevelopmentBuild)) {
		t.Fatalf("prompt %q", prompt)
	}
}

func TestPanelOpensFeedback(t *testing.T) {
	_, actions, store := fixture()
	panel := &menubar.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeightFor(store))
	settle(tt)
	if err := tt.Click(store.Copy().Text(copy.FeedbackHelp)); err != nil {
		t.Fatalf("%v %q", err, tt.Texts())
	}
	if !actions.called("feedback") {
		t.Fatalf("calls %v", actions.calls)
	}
}

func TestAboutAsksAnAgent(t *testing.T) {
	_, actions, store := fixture()
	page := settings.NewPage()
	page.Section = settings.About
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	tt.Scroll(SettingsWidth/2, SettingsHeight/2, 0, 2000)
	settle(tt)
	if err := tt.Click(store.Copy().Text(copy.AboutAskAgent)); err != nil {
		t.Fatalf("%v %q", err, tt.Texts())
	}
	if !actions.called("feedback") {
		t.Fatalf("calls %v", actions.calls)
	}
	snapshot(t, tt, "settings-about-feedback")
}
