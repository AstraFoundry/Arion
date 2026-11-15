package bootstrap

import (
	"arion.local/core/application"
	"arion.local/desktop/ui/entities/whatsnew"
	arion "arion.local/desktop/ui/shared/runtime"
)

// whatsNewOnLaunch is the release highlights a new process owes its
// profile: those of every release after the one the profile last saw, up
// to the running version. A profile still in onboarding has nothing to
// catch up on, and completing onboarding records the version it ran.
// Scripted sessions own their windows and get no dialog over them.
func whatsNewOnLaunch(prefs application.Preferences, version string, scripted bool) []arion.WhatsNewSlide {
	if scripted || !prefs.OnboardingCompleted {
		return nil
	}
	return whatsnew.Pending(prefs.WhatsNewSeen, version)
}

// presentWhatsNew hands slides to Settings, which shows them over itself
// whenever it is open: at once after an update restart, which opens
// Settings, and the next time it opens after a login-item start.
func (a *App) presentWhatsNew(slides []arion.WhatsNewSlide) {
	if len(slides) == 0 {
		return
	}
	a.state.WhatsNew = arion.WhatsNew{Slides: slides, Seq: a.state.WhatsNew.Seq + 1}
	a.invalidate()
}

// ShowWhatsNew presents the newest release's highlights on request.
func (x actions) ShowWhatsNew() { x.a.presentWhatsNew(whatsnew.Latest(x.a.version)) }

// DismissWhatsNew closes the highlights and records the running version as
// seen. A failed save is recorded by UpdatePreferences and leaves the
// stored version behind, so the next launch presents them again.
func (x actions) DismissWhatsNew() {
	a := x.a
	a.state.WhatsNew.Slides = nil
	if whatsnew.Compare(a.state.Preferences.WhatsNewSeen, a.version) < 0 {
		x.UpdatePreferences(func(p *application.Preferences) { p.WhatsNewSeen = a.version })
	}
	a.invalidate()
}
