package bootstrap

import (
	"strings"
	"testing"

	policy "arion.local/desktop/internal/productnotifications"
	arion "arion.local/desktop/ui/shared/runtime"
)

func TestProductNotificationColdActionWaitsForPermission(t *testing.T) {
	a := &App{}
	a.state.Preferences.ProductNotifications = true
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
	for _, reason := range []string{"generation", "signed-out", "disabled"} {
		t.Run(reason, func(t *testing.T) {
			a := &App{}
			a.state.Account.Phase = arion.AccountSignedIn
			a.state.Preferences.ProductNotifications = true
			generation := a.accountGeneration()
			switch reason {
			case "generation":
				a.beginAccount()
			case "signed-out":
				a.state.Account.Phase = arion.AccountSignedOut
			case "disabled":
				a.state.Preferences.ProductNotifications = false
			}
			a.finishProductNotificationAction(generation, "", nil)
			a.finishProductNotificationAction(generation, "", policy.ErrHTTP)
			if a.state.Focus.Seq != 0 || a.state.ProductNotifications.Failed {
				t.Fatal("obsolete completion changed the current account view")
			}
		})
	}
}
