package bootstrap

import (
	"errors"
	"fmt"
	"log"
	"time"

	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// Observe satisfies the core port even when local storage is unavailable.
func (a *App) Observe(d ports.Diagnostic) {
	if a.diagnostics != nil {
		a.diagnostics.Observe(d)
	}
}

func (a *App) recordEntry(e diagnostics.Entry) {
	if a.diagnostics != nil {
		a.diagnostics.Record(e)
	}
}

func (a *App) record(event string, err error) {
	if a.diagnostics != nil {
		a.diagnostics.Record(diagnostics.Entry{Event: event, Code: diagnostics.Failure(err)})
	}
}

// diagnosticExport keeps the native chooser and disk work off the UI thread.
// Its dependencies also let cancellation and failures be tested without a dialog.
type diagnosticExport struct {
	state  *arion.Diagnostics
	post   func(func())
	choose func() (string, error)
	save   func(string) error
	reveal func(string)
	record func(string, error)
}

func (x *diagnosticExport) start() {
	if x.state.Phase == arion.DiagnosticsExporting {
		return
	}
	x.state.Phase = arion.DiagnosticsExporting
	go func() {
		path, err := x.choose()
		phase := arion.DiagnosticsIdle
		event := "export_cancelled"
		if err != nil {
			phase, event = arion.DiagnosticsFailed, "export_failed"
		} else if path != "" {
			x.record("export_started", nil)
			err = x.save(path)
			phase, event = arion.DiagnosticsSaved, "export_completed"
			if err != nil {
				phase, event = arion.DiagnosticsFailed, "export_failed"
			}
		}
		x.record(event, err)
		x.post(func() {
			x.state.Phase = phase
			if phase == arion.DiagnosticsSaved {
				x.reveal(path)
			}
		})
	}()
}

func (a *App) setupDiagnosticExport() {
	a.exports = &diagnosticExport{state: &a.state.Diagnostics, post: a.post, record: a.record,
		save: func(path string) error {
			if a.diagnostics == nil {
				return errors.New("diagnostics unavailable")
			}
			return a.diagnostics.Export(path, a.version)
		},
		reveal: mygo.Shell.ShowItemInFolder}
}

func (x actions) ExportDiagnostics() {
	a := x.a
	if a.exports == nil || a.state.Diagnostics.Phase == arion.DiagnosticsExporting {
		return
	}
	messages := a.ui.Copy()
	parent := a.settingsWin
	a.exports.choose = func() (string, error) {
		return mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: parent, Title: messages.Text(copy.DiagnosticsExport),
			DefaultPath: fmt.Sprintf("Arion-logs-%s.zip", time.Now().Format("20060102-150405")),
			ButtonLabel: messages.Text(copy.DiagnosticsExport), Message: messages.Text(copy.DiagnosticsDetail),
			Filters: []mygo.FileFilter{{Name: messages.Text(copy.DiagnosticsArchive), Extensions: []string{"zip"}}}})
	}
	a.exports.start()
}

func (a *App) closeDiagnostics() {
	if a.diagnostics == nil {
		return
	}
	a.record("app_stopped", nil)
	if err := a.diagnostics.Close(); err != nil {
		// A failed log sink cannot log its own failure. Keep stderr content-free.
		log.Print("diagnostics close failed code=IO_FAILED")
	}
}
