//go:build !js && !(linux && (amd64 || arm64))

package e2e

import "github.com/egoist/mygo"

// gtkCSSBeforeRun gives Theme.SetGTKCSS a sheet that does not even parse,
// which it ignores where GTK draws nothing for the app.
func gtkCSSBeforeRun() (applied func() bool) {
	err := mygo.Theme.SetGTKCSS("button {")
	return func() bool { return err == nil }
}
