package bootstrap

import "testing"

func TestSettingsLaunch(t *testing.T) {
	cases := []struct {
		name   string
		launch settingsLaunch
		show   bool
	}{
		{name: "quit and open again", launch: settingsLaunch{OnboardingCompleted: true}, show: true},
		{name: "update restart", launch: settingsLaunch{OnboardingCompleted: true}, show: true},
		{name: "spotlight cold start", launch: settingsLaunch{OnboardingCompleted: true}, show: true},
		{name: "login item", launch: settingsLaunch{OnboardingCompleted: true, OpenedAtLogin: true}, show: false},
		{name: "onboarding still required", launch: settingsLaunch{}, show: false},
		{name: "scripted session", launch: settingsLaunch{OnboardingCompleted: true, Scripted: true}, show: false},
		{name: "notification owns the launch", launch: settingsLaunch{OnboardingCompleted: true, Notification: true}, show: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := showSettingsOnLaunch(tt.launch); got != tt.show {
				t.Fatalf("showSettingsOnLaunch() = %v, want %v", got, tt.show)
			}
		})
	}
}
