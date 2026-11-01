//go:build windows

package app

import (
	"fmt"
	"os"
	"testing"

	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/typography"
	"github.com/egoist/mygo/ui"
)

// Render the real bundled face in every Windows view test and capture.
func TestMain(m *testing.M) {
	if err := typography.Register(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// Page theme updates must retain the app-wide family.
func TestWindowsInterfaceFontSurvivesPageTheme(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, dark := range []bool{false, true} {
			name := fmt.Sprintf("%s-dark-%t", language, dark)
			t.Run(name, func(t *testing.T) {
				state, _, store := fixture()
				state.Preferences.UILanguage = language
				page := onboarding.NewPage()
				tt := ui.NewTester(interfaceView(func(c *ui.Context) {
					page.Render(c, store)
					if c.Theme().Font != typography.Family {
						t.Fatalf("page replaced interface font: %q", c.Theme().Font)
					}
				}), OnboardingWidth, OnboardingHeight)
				tt.SetDark(dark)
				settle(tt)
				snapshot(t, tt, "windows-interface-onboarding-"+name)
			})
		}
	}
}
