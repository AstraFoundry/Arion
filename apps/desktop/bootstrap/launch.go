package bootstrap

// settingsLaunch is why this process should or should not open Settings.
// MyGo reports a login-item start with WasOpenedAtLogin (the launch Apple
// Event on macOS, and --mygo-opened-at-login on Windows). A version-update
// restart uses App.Relaunch, which starts the executable again without that
// marker, so it is a user-visible launch. Spotlight, Launchpad, Finder and
// the Dock reopen a running macOS app through OnActivate
// (applicationShouldHandleReopen). The Start menu, taskbar and search start
// a second process, which OnSecondInstance delivers to the one already running.
type settingsLaunch struct {
	OnboardingCompleted bool
	OpenedAtLogin       bool
	Scripted            bool
	Notification        bool
}

// showSettingsOnLaunch reports whether a new process should open Settings.
// Login-item starts stay in the menu bar. A notification activation navigates
// on its own. Scripted sessions do not take an editor's focus.
func showSettingsOnLaunch(launch settingsLaunch) bool {
	return launch.OnboardingCompleted && !launch.OpenedAtLogin && !launch.Scripted && !launch.Notification
}
