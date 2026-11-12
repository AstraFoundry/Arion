package bootstrap

import (
	"testing"

	"arion.local/core/application"
	"arion.local/desktop/ui/entities/whatsnew"
)

func TestWhatsNewOnLaunch(t *testing.T) {
	newest := whatsnew.Releases()[0].Version
	for _, release := range whatsnew.Releases() {
		if len(release.Slides) > 0 {
			newest = release.Version
			break
		}
	}
	cases := []struct {
		name     string
		prefs    application.Preferences
		scripted bool
		show     bool
	}{
		{name: "profile from before the highlights existed", prefs: application.Preferences{OnboardingCompleted: true}, show: true},
		{name: "already seen", prefs: application.Preferences{OnboardingCompleted: true, WhatsNewSeen: newest}},
		{name: "onboarding still required", prefs: application.Preferences{}},
		{name: "scripted session", prefs: application.Preferences{OnboardingCompleted: true}, scripted: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(whatsNewOnLaunch(tt.prefs, newest, tt.scripted)) > 0; got != tt.show {
				t.Fatalf("whatsNewOnLaunch() shows = %v, want %v", got, tt.show)
			}
		})
	}
}
