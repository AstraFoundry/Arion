//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestNetworkStatus checks the reading of the connectivity hint: online
// with a route beyond the computer, as macOS and Linux, and metered as
// Microsoft's guidance says.
func TestNetworkStatus(t *testing.T) {
	const (
		unknown, none, local, internet, portal, hidden = 0, 1, 2, 3, 4, 5
		unrestricted, fixed, variable                  = 1, 2, 3
	)
	for _, tt := range []struct {
		hint nlNetworkConnectivityHint
		want platform.NetworkStatus
	}{
		{nlNetworkConnectivityHint{Level: none}, platform.NetworkStatus{}},
		{nlNetworkConnectivityHint{Level: hidden}, platform.NetworkStatus{}},
		{nlNetworkConnectivityHint{Level: unknown}, platform.NetworkStatus{Online: true}},
		{nlNetworkConnectivityHint{Level: local, Cost: unrestricted}, platform.NetworkStatus{Online: true}},
		{nlNetworkConnectivityHint{Level: portal, Cost: unrestricted}, platform.NetworkStatus{Online: true}},
		{nlNetworkConnectivityHint{Level: internet, Cost: unrestricted}, platform.NetworkStatus{Online: true}},
		{nlNetworkConnectivityHint{Level: internet, Cost: fixed}, platform.NetworkStatus{Online: true, Metered: true}},
		{nlNetworkConnectivityHint{Level: internet, Cost: variable}, platform.NetworkStatus{Online: true, Metered: true}},
		{nlNetworkConnectivityHint{Level: internet, Cost: unrestricted, Roaming: 1}, platform.NetworkStatus{Online: true, Metered: true}},
		{nlNetworkConnectivityHint{Level: internet, Cost: unrestricted, OverLimit: 1}, platform.NetworkStatus{Online: true, Metered: true}},
		{nlNetworkConnectivityHint{Level: internet, Cost: unrestricted, ApproachingLimit: 1}, platform.NetworkStatus{Online: true, Metered: true}},
	} {
		if got := networkStatus(tt.hint); got != tt.want {
			t.Errorf("networkStatus(%+v) = %+v, want %+v", tt.hint, got, tt.want)
		}
	}
}
