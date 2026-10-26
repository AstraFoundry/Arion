package bootstrap

import (
	"strings"
	"testing"

	"arion.local/desktop/internal/center"
	policy "arion.local/desktop/internal/productnotifications"
	arion "arion.local/desktop/ui/shared/runtime"
)

func TestProductNotificationColdActionWaitsForPermission(t *testing.T) {
	a := &App{}
	a.center, _ = center.Load()
	id := policy.NativePrefix + strings.Repeat("a", 64)
	for range 12 {
		a.productNotificationAction(id, "button-0")
	}
	if len(a.pendingProductActions) != 8 {
		t.Fatal("cold activation queue must be bounded while permission is unknown")
	}
	a.suspendProductNotifications()
	if len(a.pendingProductActions) != 0 {
		t.Fatal("account suspension retained old activations")
	}
}

func TestProductNotificationCompletionRejectsObsoleteAccount(t *testing.T) {
	for _, reason := range []string{"generation", "signed-out", "os-denied", "offline"} {
		t.Run(reason, func(t *testing.T) {
			a := &App{}
			a.center, _ = center.Load()
			a.state.Account.Phase = arion.AccountSignedIn
			a.state.UpdateNotifications.Permission = arion.UpdateNotificationsAllowed
			generation := a.accountGeneration()
			switch reason {
			case "generation":
				a.beginAccount()
			case "signed-out":
				a.state.Account.Phase = arion.AccountSignedOut
			case "os-denied":
				a.state.UpdateNotifications.Permission = arion.UpdateNotificationsDenied
			case "offline":
				a.opts.Offline = true
			}
			a.finishProductNotificationAction(generation, "", nil)
			a.finishProductNotificationAction(generation, "", policy.ErrHTTP)
			if a.state.Focus.Seq != 0 {
				t.Fatal("obsolete completion changed the current account view")
			}
		})
	}
}

func TestProductNotificationsAreRestrictedToStandardOnlineProfile(t *testing.T) {
	a := &App{}
	if a.productNotificationsAvailable() {
		t.Fatal("custom profile enabled account notices")
	}
	a.center, _ = center.Load()
	if !a.productNotificationsAvailable() {
		t.Fatal("standard online profile requires an obsolete preference")
	}
	a.opts.Offline = true
	if a.productNotificationsAvailable() {
		t.Fatal("offline profile enabled account notices")
	}
	a.opts.Offline = false
	a.harness = &harness{}
	if a.productNotificationsAvailable() {
		t.Fatal("E2E profile enabled account notices")
	}
}
