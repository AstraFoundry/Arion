package bootstrap

import (
	"errors"
	"testing"
	"time"

	arion "arion.local/desktop/ui/shared/runtime"
)

func TestDiagnosticExportOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, path         string
		chooseErr, saveErr error
		want               arion.DiagnosticsPhase
		saves, reveals     int
	}{
		{name: "cancel", want: arion.DiagnosticsIdle},
		{name: "chooser error", chooseErr: errors.New("chooser"), want: arion.DiagnosticsFailed},
		{name: "disk error", path: "logs.zip", saveErr: errors.New("disk"), want: arion.DiagnosticsFailed, saves: 1},
		{name: "saved", path: "logs.zip", want: arion.DiagnosticsSaved, saves: 1, reveals: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := &arion.Diagnostics{}
			posted := make(chan func(), 1)
			saves, reveals := 0, 0
			gate := make(chan struct{})
			x := &diagnosticExport{state: state, post: func(fn func()) { posted <- fn }, record: func(string, error) {},
				choose: func() (string, error) { <-gate; return tc.path, tc.chooseErr },
				save: func(path string) error {
					saves++
					if path != tc.path {
						t.Errorf("path %s", path)
					}
					return tc.saveErr
				},
				reveal: func(string) { reveals++ }}
			x.start()
			if state.Phase != arion.DiagnosticsExporting {
				t.Fatal("busy state missing")
			}
			x.start() // A second click cannot open another chooser.
			close(gate)
			select {
			case fn := <-posted:
				fn()
			case <-time.After(time.Second):
				t.Fatal("export blocked")
			}
			if state.Phase != tc.want || saves != tc.saves || reveals != tc.reveals {
				t.Fatalf("phase=%s saves=%d reveals=%d", state.Phase, saves, reveals)
			}
			select {
			case <-posted:
				t.Fatal("duplicate export")
			default:
			}
		})
	}
}
