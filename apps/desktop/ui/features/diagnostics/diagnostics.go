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
	var status copy.Key
	switch phase {
	case runtime.DiagnosticsExporting:
		label = copy.DiagnosticsExporting
	case runtime.DiagnosticsSaved:
		status = copy.DiagnosticsSaved
	case runtime.DiagnosticsFailed:
		status = copy.DiagnosticsFailed
	}
	control := func() {
		if kit.Button(c.Key("export-diagnostics"), tokens, kit.Outline, messages.Text(label), phase != runtime.DiagnosticsExporting).Clicked() {
			store.Actions.ExportDiagnostics()
		}
	}
	kit.Card(c, tokens).Children(func() {
		if status != "" {
			kit.Row(c, tokens, messages.Text(copy.DiagnosticsTitle), messages.Text(status), control)
			return
		}
		kit.TipRow(c, tokens, messages.Text(copy.DiagnosticsTitle), messages.Text(copy.DiagnosticsDetail), control)
	})
}
