//go:build arion_e2e

package bootstrap

import (
	"path/filepath"

	"arion.local/desktop/infrastructure/updatestate"
	"github.com/egoist/mygo"
)

// ShowUpdateReminderForTest hosts the production passive reminder with an
// isolated E2E state. It does not enable OS notifications or a release feed.
func (a *App) ShowUpdateReminderForTest() {
	mygo.RunOnMain(func() {
		// Offline native fixtures have no update scheduler. Explicitly own only
		// the reminder ledger in their temporary profile, without enabling a
		// feed, recovery, notifications or background update work.
		if a.updatePolicy == nil {
			a.updatePolicy = &updatePolicy{store: updatestate.Store{Path: filepath.Join(a.opts.DataDir, "updates.json")}}
		}
		a.showUpdateReminder()
	})
}
