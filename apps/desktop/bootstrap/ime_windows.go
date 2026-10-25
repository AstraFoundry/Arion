//go:build windows

package bootstrap

import (
	"arion.local/desktop/infrastructure/tsf"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

type windowsIME struct{ *tsf.IME }

func newWindowsIME(onLeave, onDictate func()) InputMethod {
	return windowsIME{tsf.New(func() (string, error) { return mygo.App.Path(mygo.PathResources) }, onLeave, onDictate)}
}
func (m windowsIME) State() runtime.InputMethodState {
	state := m.Observed()
	if state.Selected {
		return runtime.InputMethodSelected
	}
	if state.Enabled {
		return runtime.InputMethodInstalled
	}
	if state.Registered {
		return runtime.InputMethodPending
	}
	return runtime.InputMethodAbsent
}
