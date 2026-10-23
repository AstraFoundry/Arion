package macos

/*
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

type imeRemovalPending struct{}

func (imeRemovalPending) Error() string {
	return "input source removal is not independently verified; installed files retained"
}
func (imeRemovalPending) InputMethodRemovalPending() bool { return true }

var ErrIMERemovalPending error = imeRemovalPending{}

type sourceObservation struct {
	Complete      bool `json:"complete"`
	Available     bool `json:"available"`
	Selected      bool `json:"selected"`
	OwnCount      int  `json:"ownCount"`
	ParentFound   bool `json:"parentFound"`
	ParentEnabled bool `json:"parentEnabled"`
}

func (s sourceObservation) unavailable() bool { return s.Complete && !s.Available && !s.Selected }

type maintenanceResult struct {
	Version     int               `json:"version"`
	PID         int               `json:"pid"`
	Operation   string            `json:"operation"`
	Statuses    []int             `json:"statuses"`
	Observation sourceObservation `json:"observation"`
}

func observeRemoval() (sourceObservation, error) {
	var result sourceObservation
	raw := C.arion_ime_removal_observation()
	if raw == nil {
		return result, errors.New("input source observation unavailable")
	}
	defer C.free(unsafe.Pointer(raw))
	err := json.Unmarshal([]byte(C.GoString(raw)), &result)
	return result, err
}

func runIMEMaintenance(ctx context.Context, bundle, operation string) (maintenanceResult, error) {
	var result maintenanceResult
	if err := ValidateIMEPayload(bundle); err != nil {
		return result, err
	}
	path, free := cstring(bundle)
	version := C.arion_ime_maintenance_version(path)
	free()
	if version != 1 {
		return result, errors.New("input source maintenance protocol unavailable")
	}
	cmd := exec.CommandContext(ctx, filepath.Join(bundle, "Contents", "MacOS", "ArionInput"), "--arion-source-maintenance", operation)
	// The maintained host emits only bounded source state. Ignore native stderr;
	// it may contain user paths. Limit stdout before decoding the protocol.
	var output boundedMaintenanceOutput
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return result, fmt.Errorf("input source maintenance %s: %w", operation, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, err
	}
	if result.Version != 1 || result.Operation != operation || result.PID != cmd.Process.Pid {
		return result, errors.New("input source maintenance identity mismatch")
	}
	operationName := "verify_removal"
	if operation == "disable" {
		operationName = "disable"
	}
	var status int32
	for _, value := range result.Statuses {
		if value != 0 {
			status = int32(value)
			break
		}
	}
	observeInputMethodMaintenance(operationName, "after", &status, nil, &result)
	return result, nil
}

type boundedMaintenanceOutput struct{ bytes.Buffer }

func (b *boundedMaintenanceOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4096 {
		return 0, errors.New("input source maintenance output exceeds limit")
	}
	return b.Buffer.Write(p)
}

// Only observations from a new, non-mutating process can corroborate a disable.
func verifyIMERemoval(ctx context.Context, helper string) error {
	fresh, err := runIMEMaintenance(ctx, helper, "observe")
	if err != nil {
		return errors.Join(ErrIMERemovalPending, err)
	}
	local, err := observeRemoval()
	if err != nil || !fresh.Observation.unavailable() || !local.unavailable() {
		return errors.Join(ErrIMERemovalPending, err)
	}
	return nil
}

func currentInputSourceID() string {
	raw := C.arion_ime_current_id()
	if raw == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(raw))
	return C.GoString(raw)
}

func exchangeInputBundles(first, second string, existing bool) error {
	a, freeA := cstring(first)
	defer freeA()
	b, freeB := cstring(second)
	defer freeB()
	flag := C.int(0)
	if existing {
		flag = 1
	}
	if status := C.arion_ime_exchange(a, b, flag); status != 0 {
		return syscall.Errno(status)
	}
	return nil
}

func inputBundleSwapSupported(path string) bool {
	p, free := cstring(path)
	defer free()
	return C.arion_ime_swap_supported(p) != 0
}

func imeMaintenanceDirectory() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Arion", "InputMethodTransactions"), nil
}

// The stable lock file is never unlinked, so all cooperating processes lock the
// same inode. The OS releases it on exit, including an interrupted operation.
func lockIMEMaintenance() (func(), error) {
	root, err := imeMaintenanceDirectory()
	if err != nil {
		return nil, err
	}
	return lockInputMaintenance(root)
}
func lockInputMaintenance(root string) (func(), error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	fd, err := syscall.Open(filepath.Join(root, "maintenance.lock"), syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "maintenance.lock")
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, errors.New("another input source maintenance operation is active")
	}
	return func() { _ = syscall.Flock(fd, syscall.LOCK_UN); _ = file.Close() }, nil
}

// RemoveIMEWithHelper uses the shipped host's maintenance-only entry without
// starting another IMK server. The bundle stays in place on every unknown state.
func RemoveIMEWithHelper(parent context.Context, helper string) (result error) {
	imeInstallation.Lock()
	defer imeInstallation.Unlock()
	unlock, err := lockIMEMaintenance()
	if err != nil {
		return err
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	observeInputMethod("remove", "before", nil, nil)
	defer func() { observeInputMethod("remove", "after", nil, result) }()
	destination := InstalledIMEPath()
	if err := ValidateIMEPayload(helper); err != nil {
		return err
	}
	return completeInputRemoval(ctx, destination, removalOperations{
		switchAway: switchAwayIME,
		disable: func() error {
			mutation, err := runIMEMaintenance(ctx, helper, "disable")
			if err != nil {
				return errors.Join(ErrIMERemovalPending, err)
			}
			for _, status := range mutation.Statuses {
				if status != 0 {
					return fmt.Errorf("%w: disable status %d", ErrIMERemovalPending, status)
				}
			}
			return nil
		},
		verify: func() error { return verifyIMERemoval(ctx, helper) }, stop: stopInputMethod,
		stopped: func() error {
			id, free := cstring(IMEBundleID)
			defer free()
			if C.arion_ime_processes(id, 0) != 0 {
				return errors.New("input method host restarted during removal")
			}
			return nil
		},
	})
}

type removalOperations struct {
	switchAway func() error
	disable    func() error
	verify     func() error
	stop       func() error
	stopped    func() error
}

func completeInputRemoval(ctx context.Context, destination string, op removalOperations) error {
	info, err := os.Lstat(destination)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("installed input method must be a directory")
	}
	identity, err := inputBundleDigest(destination)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := op.switchAway(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := op.disable(); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrIMERemovalPending, err)
		}
		if err := op.verify(); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return errors.Join(ErrIMERemovalPending, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
	if err := op.stop(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := op.verify(); err != nil {
		return err
	}
	current, err := inputBundleDigest(destination)
	if err != nil || current != identity {
		return errors.Join(errors.New("installed input method changed during removal"), err)
	}
	return removeInputBundle(destination, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return op.stopped()
	})
}
