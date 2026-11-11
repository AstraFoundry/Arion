//go:build !(linux && (amd64 || arm64)) && !js

package e2e

// fakePortal serves the settings portal of Linux desktops, which other
// systems do not have.
func fakePortal() {}
