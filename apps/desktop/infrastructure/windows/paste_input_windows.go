//go:build windows

package windows

import "time"

func keyboardInputs(keys []pasteKey) []keyboardInput {
	inputs := make([]keyboardInput, len(keys))
	for i, key := range keys {
		inputs[i] = keyboardInput{Type: inputKeyboard, VK: key.VK, Scan: key.Scan, Flags: key.Flags, Extra: pasteInputExtra}
	}
	return inputs
}

func sendKeys(keys []pasteKey) bool {
	if len(keys) == 0 {
		return true
	}
	return send(keyboardInputs(keys))
}

// heldInterferingModifiers reads the live keyboard. Control is omitted; it
// is the paste chord.
func heldInterferingModifiers() []uint16 {
	var held []uint16
	for _, vk := range []uint16{vkMenu, vkLMenu, vkRMenu, vkShift, vkLShift, vkRShift, vkLWin, vkRWin} {
		state, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
		if state&0x8000 != 0 {
			held = append(held, vk)
		}
	}
	return held
}

// postCtrlV posts one Ctrl+V. False means the chord was not injected; the
// caller leaves the dictated text on the clipboard. Left Ctrl's key-up is
// separate and a failed key-up does not revoke a chord that was accepted.
func postCtrlV() bool {
	plan := planPaste(heldInterferingModifiers())
	if len(plan.Release) > 0 {
		if !sendKeys(plan.Release) {
			return false
		}
		time.Sleep(pastePhaseGap)
	}
	if !sendKeys(plan.Chord) {
		return false
	}
	time.Sleep(pastePhaseGap)
	sendKeys(plan.CtrlUp)
	return true
}
