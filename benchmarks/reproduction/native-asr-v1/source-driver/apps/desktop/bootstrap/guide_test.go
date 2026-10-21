package bootstrap

import (
	"testing"

	arion "arion.local/desktop/ui/shared/runtime"

	"github.com/egoist/mygo"
)

func TestGuideBounds(t *testing.T) {
	area := mygo.Rectangle{X: 0, Y: 25, Width: 1512, Height: 920}
	cases := []struct {
		name     string
		settings mygo.Rectangle
		want     mygo.Rectangle
	}{
		{"under the pane", mygo.Rectangle{X: 300, Y: 120, Width: 715, Height: 600},
			mygo.Rectangle{X: 530, Y: 728, Width: 485, Height: 112}},
		{"narrow window keeps a readable width", mygo.Rectangle{X: 300, Y: 120, Width: 500, Height: 600},
			mygo.Rectangle{X: 530, Y: 728, Width: guideMinWidth, Height: 112}},
		{"tall window overlaps its foot", mygo.Rectangle{X: 300, Y: 40, Width: 715, Height: 880},
			mygo.Rectangle{X: 530, Y: 25 + 920 - guideInset - 112, Width: 485, Height: 112}},
		{"window past the right edge", mygo.Rectangle{X: 1300, Y: 120, Width: 715, Height: 600},
			mygo.Rectangle{X: 1512 - guideInset - 485, Y: 728, Width: 485, Height: 112}},
		{"window past the left edge", mygo.Rectangle{X: -400, Y: 120, Width: 715, Height: 600},
			mygo.Rectangle{X: guideInset, Y: 728, Width: 485, Height: 112}},
	}
	for _, tc := range cases {
		if got := guideBounds(tc.settings, area, 112); got != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestGuideBoundsSecondaryDisplay(t *testing.T) {
	area := mygo.Rectangle{X: 1512, Y: -200, Width: 1920, Height: 1055}
	got := guideBounds(mygo.Rectangle{X: 1700, Y: -100, Width: 715, Height: 600}, area, 112)
	want := mygo.Rectangle{X: 1930, Y: 508, Width: 485, Height: 112}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// A copied or registered-but-disabled source must keep the settings guide open.
func TestInputMethodGuideRequiresEnabledSource(t *testing.T) {
	for _, state := range []arion.InputMethodState{arion.InputMethodAbsent, arion.InputMethodPending, arion.InputMethodUnsupported} {
		if inputMethodGuideReady(state) {
			t.Fatalf("guide completed before enablement: %s", state)
		}
	}
	for _, state := range []arion.InputMethodState{arion.InputMethodInstalled, arion.InputMethodSelected} {
		if !inputMethodGuideReady(state) {
			t.Fatalf("guide failed to acknowledge enablement: %s", state)
		}
	}
}

type guidePermissionPlatform struct {
	Platform
	microphone, screen, accessibility arion.Consent
	requests, settings                int
}

func (p *guidePermissionPlatform) Microphone() arion.Consent      { return p.microphone }
func (p *guidePermissionPlatform) ScreenRecording() arion.Consent { return p.screen }
func (p *guidePermissionPlatform) Accessibility() arion.Consent   { return p.accessibility }
func (p *guidePermissionPlatform) AppBundle() string              { return "" }
func (p *guidePermissionPlatform) RequestMicrophone()             { p.requests++ }
func (p *guidePermissionPlatform) OpenMicrophoneSettings()        { p.settings++ }
func (p *guidePermissionPlatform) RequestScreenRecording()        { p.requests++ }

func TestSwitchGuidesReadTheirOwnPermission(t *testing.T) {
	p := &guidePermissionPlatform{microphone: arion.ConsentDenied, screen: arion.ConsentDenied, accessibility: arion.ConsentGranted}
	a := &App{platform: p}
	if a.guideReady(arion.GuideMicrophone) || a.guideReady(arion.GuideScreenRecording) {
		t.Fatal("unrelated Accessibility consent completed another permission guide")
	}
	p.microphone = arion.ConsentGranted
	if !a.guideReady(arion.GuideMicrophone) || a.guideReady(arion.GuideScreenRecording) {
		t.Fatal("microphone consent did not remain independent of screen consent")
	}
	p.screen = arion.ConsentGranted
	if !a.guideReady(arion.GuideScreenRecording) {
		t.Fatal("screen grant was not acknowledged")
	}
}

func TestPermissionRequestsPreserveNativePromptsWithoutGuideHost(t *testing.T) {
	p := &guidePermissionPlatform{screen: arion.ConsentDenied}
	x := actions{&App{platform: p}}
	x.RequestMicrophone()
	if p.requests != 1 || p.settings != 0 {
		t.Fatal("initial microphone request navigated instead of prompting")
	}
	x.OpenMicrophoneSettings()
	if p.requests != 1 || p.settings != 1 {
		t.Fatal("microphone settings did not keep the unbundled platform action")
	}
	x.RequestScreenRecording()
	if p.requests != 2 {
		t.Fatal("screen guide replaced or duplicated the native permission request")
	}
}
