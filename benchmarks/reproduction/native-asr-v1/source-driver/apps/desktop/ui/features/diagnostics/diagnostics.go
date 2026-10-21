// Package diagnostics owns the local support-log export action in About.
package diagnostics

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func Card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	phase := store.State.Diagnostics.Phase
	label := copy.DiagnosticsExport
	detail := copy.DiagnosticsDetail
	switch phase {
	case runtime.DiagnosticsExporting:
		label = copy.DiagnosticsExporting
	case runtime.DiagnosticsSaved:
		detail = copy.DiagnosticsSaved
	case runtime.DiagnosticsFailed:
		detail = copy.DiagnosticsFailed
	}
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.DiagnosticsTitle), messages.Text(detail), func() {
			if kit.Button(c.Key("export-diagnostics"), tokens, kit.Outline, messages.Text(label), phase != runtime.DiagnosticsExporting).Clicked() {
				store.Actions.ExportDiagnostics()
			}
		})
	})
}
