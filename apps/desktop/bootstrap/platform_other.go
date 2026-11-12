//go:build !darwin && !windows

package bootstrap

import (
	"context"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/infrastructure/holdkey"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// otherPlatform is the Linux checkout path. It keeps the shared shell
// testable. It does not observe hold keys and does not install an input method.
type otherPlatform struct{}

func newPlatform() Platform { return otherPlatform{} }

// selectionReader is false: no host reads another app's selected text here,
// so onboarding omits the selection page.
const selectionReader = false

func configureInputMethodDiagnostics(func(diagnostics.Entry)) {}
func configureClipboardWindow(*mygo.Window)                   {}
func configureClipboardAccessWindow(*mygo.Window)             {}
func configureOverlay(*mygo.Window)                           {}

func (otherPlatform) Microphone() runtime.Consent               { return runtime.ConsentUndetermined }
func (otherPlatform) Accessibility() runtime.Consent            { return runtime.ConsentUndetermined }
func (otherPlatform) MicrophoneAuthorized() bool                { return false }
func (otherPlatform) RequestMicrophone()                        {}
func (otherPlatform) RequestAccessibility()                     {}
func (otherPlatform) ScreenRecording() runtime.Consent          { return runtime.ConsentUndetermined }
func (otherPlatform) RequestScreenRecording()                   {}
func (otherPlatform) OpenMicrophoneSettings()                   {}
func (otherPlatform) OpenAccessibilitySettings()                {}
func (otherPlatform) InputMonitoring() runtime.Consent          { return runtime.ConsentUndetermined }
func (otherPlatform) RequestInputMonitoring()                   {}
func (otherPlatform) AppBundle() string                         { return "" }
func (otherPlatform) OpenGuidePane(runtime.GuidePane)           {}
func (otherPlatform) SettingsWindow() SettingsWindow            { return SettingsWindow{} }
func (otherPlatform) StopHoldKey()                              {}
func (otherPlatform) NewInputMethod(func(), func()) InputMethod { return nil }

func (otherPlatform) StartHoldKeys([]application.HoldKey, func(application.HoldKey, holdkey.Event)) error {
	return domain.Fail(domain.CodeTargetUnsupported, "hold key")
}

func (otherPlatform) ReadSelection() (string, bool, error) {
	return "", false, domain.Fail(domain.CodeTargetUnsupported, "selection")
}
func (otherPlatform) SelectionAnchor() (int, int, int, int, bool) { return 0, 0, 0, 0, false }
func (otherPlatform) SelectionSource() application.SelectionSource {
	return application.SelectionSource{}
}
func (otherPlatform) StartSelectionWatch(func()) error {
	return domain.Fail(domain.CodeTargetUnsupported, "selection")
}
func (otherPlatform) StopSelectionWatch() {}

func (otherPlatform) NewTargets(func() application.Delivery, TargetGates) Targets {
	return otherTargets{}
}

// otherTargets refuses every read and insert. There is no focused-field host.
type otherTargets struct{}

func (otherTargets) Capture(context.Context, contracts.Identity) (contracts.TargetSnapshot, error) {
	return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "target")
}
func (otherTargets) ReadContext(context.Context, contracts.Identity, contracts.TargetToken) (contracts.ContextSnapshot, error) {
	return contracts.ContextSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "target")
}
func (otherTargets) Commit(context.Context, contracts.CommitRequest) (contracts.CommitResult, error) {
	return contracts.CommitResult{}, domain.Fail(domain.CodeTargetUnsupported, "target")
}
func (otherTargets) Follow(contracts.TargetToken) bool { return false }
func (otherTargets) ReadField(context.Context, contracts.TargetToken) (ports.FieldRead, error) {
	return ports.FieldRead{}, domain.Fail(domain.CodeTargetUnsupported, "target")
}
func (otherTargets) Release(contracts.TargetToken) {}
func (otherTargets) Discard(contracts.TargetToken) {}
func (otherTargets) FocusOwner() int               { return 0 }
func (otherTargets) Peek(context.Context, int) (contracts.TargetSnapshot, error) {
	return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "target")
}
