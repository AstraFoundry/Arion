//go:build darwin

package macos

/*
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/imebridge"
)

var imeInstallation sync.Mutex

// IMESourceID is the input source Arion's IMK bundle declares.
const IMESourceID = "si.astrafoundry.inputmethod.arion.pinyin"

// IMEState is the registration state of Arion's input source.
type IMEState int

const (
	IMEAbsent IMEState = iota
	IMERegistered
	IMEEnabled
	IMESelected
)

func cstring(s string) (*C.char, func()) {
	c := C.CString(s)
	return c, func() { C.free(unsafe.Pointer(c)) }
}

// InstalledIMEPath is where a user's input methods live.
func InstalledIMEPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Input Methods", "ArionInput.app")
}

func inputBundleMatches(bundle string) bool {
	path, freePath := cstring(bundle)
	defer freePath()
	id, freeID := cstring(IMESourceID)
	defer freeID()
	return C.arion_ime_bundle_matches(path, id) != 0
}

// QueryIME reports the input source state.
func QueryIME() IMEState {
	if info, err := os.Stat(InstalledIMEPath()); err != nil || !info.IsDir() {
		return IMEAbsent
	}
	id, free := cstring(IMESourceID)
	defer free()
	if !inputBundleMatches(InstalledIMEPath()) {
		return IMEAbsent
	}
	return IMEState(C.arion_ime_state(id))
}

// InstallIME copies the bundled input method into ~/Library/Input Methods,
// registers and enables it. The user selects it in the input menu (IME-01).
func InstallIME(bundled string) error {
	imeInstallation.Lock()
	defer imeInstallation.Unlock()
	if !inputBundleMatches(bundled) {
		return domain.Fail(domain.CodeTargetUnsupported, "input bundle does not declare the Arion input source")
	}
	destination := InstalledIMEPath()
	previous := QueryIME()
	register := func() error {
		path, free := cstring(destination)
		defer free()
		if status := C.arion_ime_register(path); status != 0 {
			return domain.Fail(domain.CodeTargetUnsupported, fmt.Sprintf("input source registration failed: %d", status))
		}
		return nil
	}
	requestEnable := func() int {
		id, free := cstring(IMESourceID)
		defer free()
		return int(C.arion_ime_enable(id))
	}
	state := func() IMEState {
		id, free := cstring(IMESourceID)
		defer free()
		return IMEState(C.arion_ime_state(id))
	}
	return replaceInputBundle(bundled, destination, func() error {
		return registerAndEnableInputSource(register, requestEnable, state, false)
	}, func() error {
		if previous == IMEAbsent {
			return disableIME()
		}
		if err := register(); err != nil {
			return err
		}
		if previous >= IMEEnabled {
			return enableInputSource(requestEnable, state, true)
		}
		return disableIME()
	})
}

// Registration can finish while macOS still requires the user to enable the
// source in Keyboard settings. Retain that registered bundle as pending.
func registerAndEnableInputSource(register func() error, enable func() int, state func() IMEState, requireEnabled bool) error {
	if err := register(); err != nil {
		return err
	}
	if current := state(); current < IMERegistered || current > IMESelected {
		return domain.Fail(domain.CodeTargetUnsupported, "input source was not discovered after registration")
	}
	return enableInputSource(enable, state, requireEnabled)
}

func enableInputSource(enable func() int, state func() IMEState, requireEnabled bool) error {
	if status := enable(); status != 0 {
		return domain.Fail(domain.CodeTargetUnsupported, fmt.Sprintf("input source enable failed: %d", status))
	}
	current := state()
	if current < IMERegistered || current > IMESelected {
		return domain.Fail(domain.CodeTargetUnsupported, "input source was not discovered after enable request")
	}
	if requireEnabled && current < IMEEnabled {
		return domain.Fail(domain.CodeTargetUnsupported, "previous input source remains disabled after restoration")
	}
	return nil
}

// OpenIMEKeyboardSettings opens the system's Input Sources preferences.
func OpenIMEKeyboardSettings() error {
	if C.arion_ime_open_settings() == 0 {
		return domain.Fail(domain.CodeTargetUnsupported, "keyboard input source settings could not be opened")
	}
	return nil
}

func disableIME() error {
	id, free := cstring(IMESourceID)
	defer free()
	if status := C.arion_ime_disable(id); status != 0 {
		return domain.Fail(domain.CodeTargetUnsupported, fmt.Sprintf("input source disable failed: %d", status))
	}
	return nil
}

// RemoveIME changes away from this source before disabling and deleting it.
func RemoveIME() error {
	imeInstallation.Lock()
	defer imeInstallation.Unlock()
	return removeInputBundle(InstalledIMEPath(), disableIME)
}

// IME is the session-bound local transport used by the InputMethodKit host.
type IME = imebridge.IME

func NewIME(path string, onLeave func()) *IME {
	return imebridge.New(path, onLeave)
}

// DefaultIMESocket matches the path the IMK bundle dials.
func DefaultIMESocket() string {
	base, _ := os.UserConfigDir()
	return filepath.Join(base, "Arion", "ime.sock")
}
