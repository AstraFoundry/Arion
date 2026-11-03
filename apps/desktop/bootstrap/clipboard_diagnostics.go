package bootstrap

import (
	"arion.local/desktop/infrastructure/clipboardos"
	"arion.local/desktop/infrastructure/diagnostics"
)

// Clipboard providers may expose bounded metadata without editor content.
// Keep platform-specific query details out of the portable target contract.
func (a *App) recordClipboardTarget(target clipboardos.Target) {
	if a.diagnostics == nil {
		return
	}
	if source, ok := target.(interface {
		ClipboardDiagnostic() diagnostics.ClipboardTargetDiagnostic
	}); ok {
		d := source.ClipboardDiagnostic()
		a.recordEntry(diagnostics.Entry{Event: "clipboard_target", ClipboardTarget: &d})
		return
	}
	if target == nil {
		d := diagnostics.ClipboardTargetDiagnostic{
			Focus: "not_queried", Window: "not_queried", Verification: "no_destination",
		}
		a.recordEntry(diagnostics.Entry{Event: "clipboard_target", ClipboardTarget: &d})
	}
}
