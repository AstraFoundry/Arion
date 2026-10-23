package macos

/*
#include "native.h"
*/
import "C"

import (
	"os"
	"sync"

	"arion.local/desktop/infrastructure/diagnostics"
)

var inputDiagnostics struct {
	sync.Mutex
	record func(diagnostics.Entry)
	last   diagnostics.InputMethodDiagnostic
	seen   bool
}

// SetIMEDiagnostics connects process-local TIS observations to the bounded log.
// Passing nil detaches it. It never changes source registration or selection.
func SetIMEDiagnostics(record func(diagnostics.Entry)) {
	inputDiagnostics.Lock()
	defer inputDiagnostics.Unlock()
	inputDiagnostics.record = record
	inputDiagnostics.seen = false
}

func inputSourceDiagnostic(id string, flags C.int) diagnostics.InputSourceDiagnostic {
	return diagnostics.InputSourceDiagnostic{ID: id, Found: flags&1 != 0, Enabled: flags&2 != 0,
		Selected: flags&4 != 0, Selectable: flags&8 != 0, IdentityMatches: flags&16 != 0}
}

func observeInputMethod(operation, step string, status *int32, failure error) {
	observeInputMethodMaintenance(operation, step, status, failure, nil)
}
func observeInputMethodMaintenance(operation, step string, status *int32, failure error, maintenance *maintenanceResult) {
	inputDiagnostics.Lock()
	active := inputDiagnostics.record != nil
	inputDiagnostics.Unlock()
	if !active {
		return
	}
	id, freeID := cstring(IMESourceID)
	defer freeID()
	parent, freeParent := cstring(IMEBundleID)
	defer freeParent()
	var native C.arion_ime_snapshot
	C.arion_ime_inspect(id, parent, &native)
	current := "unavailable"
	switch native.selected_kind {
	case 1:
		current = "arion_mode"
	case 2:
		current = "arion_parent"
	case 3:
		current = "other"
	}
	info, err := os.Stat(InstalledIMEPath())
	d := diagnostics.InputMethodDiagnostic{Operation: operation, Step: step, OSStatus: status, ProcessID: os.Getpid(),
		Installed: err == nil && info.IsDir(), Current: current,
		Mode: inputSourceDiagnostic(IMESourceID, native.mode_flags), Parent: inputSourceDiagnostic(IMEBundleID, native.parent_flags)}
	if maintenance != nil {
		d.MaintenancePID = maintenance.PID
		d.MaintenanceState = "unknown"
		if maintenance.Observation.Complete {
			d.MaintenanceState = "unavailable"
			if maintenance.Observation.Available {
				d.MaintenanceState = "available"
			}
			if maintenance.Observation.Selected {
				d.MaintenanceState = "selected"
			}
		}
	}
	if err != nil && !os.IsNotExist(err) {
		d.BundleCode = diagnostics.Failure(err)
	}
	switch operation {
	case "select", "enable", "disable":
		d.RequestedID = IMESourceID
	case "register":
		d.RequestedID = IMEBundleID
	}
	if operation != "query" && d.Installed {
		// Missing/unreadable executable has no digest; installed and found flags
		// remain independent, and the actual operation reports its own failure.
		var buildErr error
		d.HostBuild, buildErr = inputBuildIdentity(InstalledIMEPath())
		d.BuildCode = diagnostics.Failure(buildErr)
	}
	emitInputMethodDiagnostic(d, failure)
}

func emitInputMethodDiagnostic(d diagnostics.InputMethodDiagnostic, failure error) {
	inputDiagnostics.Lock()
	if d.Operation == "query" {
		if inputDiagnostics.seen && inputDiagnostics.last == d {
			inputDiagnostics.Unlock()
			return
		}
		inputDiagnostics.last, inputDiagnostics.seen = d, true
	}
	record := inputDiagnostics.record
	inputDiagnostics.Unlock()
	if record != nil {
		record(diagnostics.Entry{Event: "ime_native", InputMethod: &d, Code: diagnostics.Failure(failure)})
	}
}

func traceInputMethodNative(operation string, call func() C.int) C.int {
	observeInputMethod(operation, "before", nil, nil)
	result := call()
	status := int32(result)
	observeInputMethod(operation, "after", &status, nil)
	return result
}
