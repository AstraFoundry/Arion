//go:build mygo_noinspector && !js

package e2e

// Builds without the inspector leave plugins/automation out.
func automationApp() {}
