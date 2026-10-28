package application

import (
	"slices"
	"strconv"
	"strings"
)

// KeyModifiers are the required modifier families of a dictation shortcut.
// Either physical side satisfies a family. A shortcut matches exactly.
type KeyModifiers uint8

const (
	KeyCtrl KeyModifiers = 1 << iota
	KeyAlt
	KeyShift
	KeySuper
)

// Shortcut keeps the trigger separate from the modifiers that precede it.
type Shortcut struct {
	Modifiers KeyModifiers
	Key       HoldKey
}

// ShortcutKeysFor lists non-modifier triggers. Text/navigation keys require
// a modifier, while function keys can be used alone.
func ShortcutKeysFor(platform string) []HoldKey {
	if DefaultHoldKey(platform) == "" {
		return nil
	}
	keys := []HoldKey{"space", "enter", "tab", "backspace", "delete", "home", "end", "page_up", "page_down", "left", "right", "up", "down", "minus", "equal", "comma", "period", "slash", "semicolon", "quote", "bracket_left", "bracket_right", "backslash", "grave"}
	for ch := 'a'; ch <= 'z'; ch++ {
		keys = append(keys, HoldKey(string(ch)))
	}
	for ch := '0'; ch <= '9'; ch++ {
		keys = append(keys, HoldKey(string(ch)))
	}
	n := 20
	if platform == "windows" {
		n = 24
	}
	for i := 1; i <= n; i++ {
		keys = append(keys, HoldKey("f"+strconv.Itoa(i)))
	}
	return keys
}

// HoldShortcut constructs a canonical persisted key and rejects ordinary
// unmodified typing, unsupported keys and unknown modifier bits.
func HoldShortcut(platform string, mods KeyModifiers, key HoldKey) (HoldKey, bool) {
	if mods & ^(KeyCtrl|KeyAlt|KeyShift|KeySuper) != 0 {
		return "", false
	}
	if mods == 0 && slices.Contains(HoldKeysFor(platform), key) {
		return key, true
	}
	if !slices.Contains(ShortcutKeysFor(platform), key) {
		return "", false
	}
	if mods == 0 && !isFunctionKey(key) {
		return "", false
	}
	parts := []string{}
	for _, m := range []struct {
		bit  KeyModifiers
		name string
	}{{KeyCtrl, "ctrl"}, {KeyAlt, "alt"}, {KeyShift, "shift"}, {KeySuper, "super"}} {
		if mods&m.bit != 0 {
			parts = append(parts, m.name)
		}
	}
	return HoldKey(strings.Join(append(parts, string(key)), "+")), true
}

// ParseHoldShortcut accepts only the canonical format written by the UI;
// malformed profiles are handled by Preferences.Normalize.
func ParseHoldShortcut(platform string, key HoldKey) (Shortcut, bool) {
	parts := strings.Split(string(key), "+")
	var mods KeyModifiers
	for _, part := range parts[:len(parts)-1] {
		var bit KeyModifiers
		switch part {
		case "ctrl":
			bit = KeyCtrl
		case "alt":
			bit = KeyAlt
		case "shift":
			bit = KeyShift
		case "super":
			bit = KeySuper
		default:
			return Shortcut{}, false
		}
		if mods&bit != 0 {
			return Shortcut{}, false
		}
		mods |= bit
	}
	s := Shortcut{Modifiers: mods, Key: HoldKey(parts[len(parts)-1])}
	canonical, ok := HoldShortcut(platform, s.Modifiers, s.Key)
	return s, ok && canonical == key
}

func isFunctionKey(key HoldKey) bool {
	n, err := strconv.Atoi(strings.TrimPrefix(string(key), "f"))
	return err == nil && strings.HasPrefix(string(key), "f") && n >= 1 && n <= 24
}

// HoldKeysConflict rejects only identical enabled assignments. A standalone
// modifier and a chord using its family coexist through chord-first activation.
func HoldKeysConflict(a, b HoldKey) bool {
	return a != "" && a != HoldOff && a == b
}
