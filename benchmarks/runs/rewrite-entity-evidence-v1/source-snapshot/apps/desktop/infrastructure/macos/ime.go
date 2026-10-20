//go:build darwin

package macos

/*
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/imebridge"
)

var imeInstallation sync.Mutex

// IMESourceID is the input source Arion's IMK bundle declares.
const IMESourceID = "si.astrafoundry.inputmethod.arion.pinyin"

// IMEBundleID is the IMK bundle's own identifier.
const IMEBundleID = "si.astrafoundry.inputmethod.arion"

// IMEState is the registration state of Arion's input source.
type IMEState int

const (
	IMEAbsent IMEState = iota
	IMERegistered
	IMEEnabled
	IMESelected
	// IMECopied describes valid installed bytes not yet discovered by TIS.
	// Native arion_ime_state retains its independent 0..3 contract.
	IMECopied
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
	return queryInstalledIME(InstalledIMEPath(), func() IMEState {
		id, free := cstring(IMESourceID)
		defer free()
		return IMEState(C.arion_ime_state(id))
	})
}

func queryInstalledIME(bundle string, state func() IMEState) IMEState {
	if info, err := os.Stat(bundle); err != nil || !info.IsDir() || !inputBundleMatches(bundle) {
		return IMEAbsent
	}
	if current := state(); current != IMEAbsent {
		return current
	}
	return IMECopied
}

// ValidateIMEPayload verifies Developer ID signature and default Data Protection
// identity consistency. Release packaging separately binds the configured
// publisher and validates provisioning authorization; this is not a team pin.
func ValidateIMEPayload(bundle string) error {
	if !inputBundleMatches(bundle) {
		return domain.Fail(domain.CodeArtifactUnverified, "input method payload metadata does not match Arion")
	}
	path, freePath := cstring(bundle)
	defer freePath()
	identifier, freeID := cstring(IMEBundleID)
	defer freeID()
	if status := C.arion_ime_validate_payload(path, identifier); status != 0 {
		return domain.Fail(domain.CodeArtifactUnverified, fmt.Sprintf("input method payload signature or provisioning identity is invalid: %d", status))
	}
	return nil
}

// InstallIME copies the bundled input method into ~/Library/Input Methods,
// registers and enables it. The user selects it in the input menu (IME-01).
func InstallIME(bundled string) error {
	imeInstallation.Lock()
	defer imeInstallation.Unlock()
	if err := ValidateIMEPayload(bundled); err != nil {
		return err
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
	err := replaceInputBundle(bundled, destination, func() error {
		return registerAndEnableInputSource(register, requestEnable, state, false)
	}, func() error {
		if previous == IMEAbsent {
			return disableIME()
		}
		if previous == IMECopied {
			return disableIME()
		}
		if err := register(); err != nil {
			return err
		}
		if previous == IMEEnabled || previous == IMESelected {
			return enableInputSource(requestEnable, state, true)
		}
		return disableIME()
	})
	if err == nil && previous != IMEAbsent {
		// A replaced input method keeps running its old binary until its
		// process ends; macOS launches the new one on the next activation.
		id, free := cstring(IMEBundleID)
		defer free()
		C.arion_ime_terminate(id)
	}
	return err
}

// IMEUpdateAvailable reports whether the installed input method differs from
// the bundled one. The executable and the signature's resource list
// (_CodeSignature/CodeResources, which hashes every resource) identify a
// build without reading the whole bundle.
func IMEUpdateAvailable(bundled string) bool {
	return bundleDiffers(InstalledIMEPath(), bundled)
}

func bundleDiffers(installed, bundled string) bool {
	if _, err := os.Stat(installed); err != nil {
		return false
	}
	for _, name := range []string{"Contents/MacOS/ArionInput", "Contents/_CodeSignature/CodeResources", "Contents/Info.plist"} {
		a, errA := os.ReadFile(filepath.Join(installed, name))
		b, errB := os.ReadFile(filepath.Join(bundled, name))
		if errB != nil {
			return false
		}
		if errA != nil || !bytes.Equal(a, b) {
			return true
		}
	}
	return false
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

// WriteKeyboardSettings writes keyboard.json beside the socket, where the
// IMK bundle reads it on activation.
func WriteKeyboardSettings(k application.KeyboardPreferences) error {
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	return imebridge.WriteKeyboardSettings(filepath.Join(base, "Arion", "keyboard.json"), k)
}

// DefaultIMESocket matches the path the IMK bundle dials.
func DefaultIMESocket() string {
	base, _ := os.UserConfigDir()
	return filepath.Join(base, "Arion", "ime.sock")
}
