package app

import (
	"testing"

	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestOnboardingFlow(t *testing.T) {
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = runtime.UILanguage(locale)
			state.Permissions = runtime.Permissions{Microphone: runtime.ConsentUndetermined, Accessibility: runtime.ConsentUndetermined}
			state.Model.State = runtime.ModelAwaitingDownload
			page := onboarding.NewPage()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
			settle(tt)
			messages := store.Copy()
			click := func(key copy.Key) { t.Helper(); clicker(t, tt)(messages.Text(key)) }
			snapshot(t, tt, "onboarding-welcome-"+string(locale))
			click(copy.OnbStart)
			expect(t, tt, true, messages.Text(copy.OnbModelTitle), messages.DownloadSize(state.Model.Active.SizeBytes))
			if actions.called("retry") {
				t.Fatal("opening setup downloaded a model")
			}
			download, ok := tt.Find(messages.Text(copy.OnbDownload))
			title, titleOK := tt.Find(messages.Text(copy.ModelNameGeneric))
			if !ok || !titleOK || download.W >= 180 || download.X < title.X+200 || download.Y > title.Y+title.H {
				t.Fatal("download action must stay compact at the model header's trailing edge")
			}
			snapshot(t, tt, "onboarding-download-"+string(locale))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbModelTitle))
			click(copy.OnbDownload)
			if !actions.called("retry") {
				t.Fatal("download confirmation did not start provisioning")
			}
			state.Model.State = runtime.ModelProvisioning
			settle(tt)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbMicTitle))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbMicTitle))
			click(copy.OnbMicAllow)
			if !actions.called("request-mic") {
				t.Fatal("microphone not requested")
			}
			state.Permissions.Microphone = runtime.ConsentGranted
			settle(tt)
			snapshot(t, tt, "onboarding-microphone-"+string(locale))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbAXTitle))
			snapshot(t, tt, "onboarding-accessibility-"+string(locale))
			// External consent is observed before continuing.
			state.Permissions.Accessibility = runtime.ConsentGranted
			settle(tt)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbTryTitle))
			expect(t, tt, false, messages.Text(copy.OnbTryPlaceholder))
			snapshot(t, tt, "onboarding-practice-downloading-"+string(locale))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbTryTitle))
			state.Model.State = runtime.ModelFailed
			settle(tt)
			click(copy.Retry)
			state.Model.State = runtime.ModelReady
			state.Hold = runtime.Hold{Failure: runtime.CodePermissionDenied}
			settle(tt)
			snapshot(t, tt, "onboarding-practice-permission-"+string(locale))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbTryTitle))
			state.Hold = runtime.Hold{Active: true}
			settle(tt)
			expect(t, tt, true, messages.Text(copy.OnbTryPlaceholder), messages.Text(copy.HoldKeyEdit))
			snapshot(t, tt, "onboarding-practice-"+string(locale))
			state.Dictation.State = runtime.StateFinalizing
			settle(tt)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbTryTitle))
			state.Dictation.State = runtime.StateCompleted
			settle(tt)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbDoneTitle))
			snapshot(t, tt, "onboarding-complete-"+string(locale))
			// Losing readiness must restore actionable controls, even on the last page.
			state.Permissions.Microphone = runtime.ConsentDenied
			settle(tt)
			expect(t, tt, true, messages.Text(copy.OnbTryTitle), messages.Text(copy.OpenSettings))
			state.Permissions.Microphone = runtime.ConsentGranted
			settle(tt)
			click(copy.Continue)
			click(copy.OnbFinish)
			if !actions.called("complete") {
				t.Fatal("onboarding not completed")
			}
			for _, call := range []string{"sign-in", "install-ime", "download-polish", "check-stored"} {
				if actions.called(call) {
					t.Fatalf("required flow triggered optional action %s", call)
				}
			}
		})
	}
}
