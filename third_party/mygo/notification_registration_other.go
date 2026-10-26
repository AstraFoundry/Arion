//go:build !windows || (!amd64 && !arm64)

package mygo

import "github.com/egoist/mygo/internal/platform"

// HandleNotificationRegistration is the Windows installer entry point. Other
// platforms do not need desktop COM notification registration.
func HandleNotificationRegistration() (handled bool, err error) { return false, nil }

func notificationRelayRequested() bool { return false }
func relayNotificationActivation(forward func(id, action string) error) error {
	return platform.ErrUnsupported
}
