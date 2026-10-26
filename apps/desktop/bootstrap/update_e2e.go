//go:build arion_e2e

package bootstrap

import "github.com/egoist/mygo"

// ShowUpdateReminderForTest hosts the production passive reminder with an
// isolated E2E state. It does not enable OS notifications or a release feed.
func (a *App) ShowUpdateReminderForTest() {
	mygo.RunOnMain(a.showUpdateReminder)
}
