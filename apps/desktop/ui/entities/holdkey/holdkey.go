// Package holdkey names the dictation key, the only global activation.
package holdkey

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	platform "runtime"
	"strings"
)

var names = map[runtime.HoldKey]copy.Key{
	runtime.HoldOff:         copy.HoldKeyOff,
	runtime.HoldLeftOption:  copy.HoldKeyLeftOption,
	runtime.HoldLeftCommand: copy.HoldKeyLeftCommand,
	runtime.HoldLeftCtrl:    copy.HoldKeyLeftCtrl,
	runtime.HoldLeftAlt:     copy.HoldKeyLeftAlt,
	runtime.HoldLeftShift:   copy.HoldKeyLeftShift,
	runtime.HoldRightShift:  copy.HoldKeyRightShift,
	runtime.HoldMenu:        copy.HoldKeyMenu,
	"space":                 copy.HoldKeySpace, "enter": copy.HoldKeyEnter, "tab": copy.HoldKeyTab,
	"backspace": copy.HoldKeyBackspace, "delete": copy.HoldKeyDelete,
	"home": copy.HoldKeyHome, "end": copy.HoldKeyEnd, "page_up": copy.HoldKeyPageUp, "page_down": copy.HoldKeyPageDown,
	"left": copy.HoldKeyLeft, "right": copy.HoldKeyRight, "up": copy.HoldKeyUp, "down": copy.HoldKeyDown,
	runtime.HoldFn:           copy.HoldKeyFn,
	runtime.HoldRightOption:  copy.HoldKeyRightOption,
	runtime.HoldRightCommand: copy.HoldKeyRightCommand,
	runtime.HoldRightAlt:     copy.HoldKeyRightAlt,
	runtime.HoldRightCtrl:    copy.HoldKeyRightCtrl,
}

// Physical punctuation legends retain their identity in both locales.
var punctuation = map[runtime.HoldKey]string{
	"minus": "-", "equal": "=", "comma": ",", "period": ".", "slash": "/", "semicolon": ";", "quote": "'", "bracket_left": "[", "bracket_right": "]", "backslash": "\\", "grave": "`",
}

// Name is the key's display name.
func Name(messages copy.Catalog, key runtime.HoldKey) string {
	if key == "" {
		key = runtime.HoldOff
	}
	if name, ok := names[key]; ok {
		return messages.Text(name)
	}
	parts := strings.Split(string(key), "+")
	for i, part := range parts {
		switch part {
		case "ctrl":
			parts[i] = "Ctrl"
		case "alt":
			parts[i] = AltName()
		case "shift":
			parts[i] = "Shift"
		case "super":
			parts[i] = SuperName()
		default:
			if name, ok := names[runtime.HoldKey(part)]; ok {
				parts[i] = messages.Text(name)
			} else {
				if legend, ok := punctuation[runtime.HoldKey(part)]; ok {
					parts[i] = legend
				} else {
					parts[i] = strings.ToUpper(part)
				}
			}
		}
	}
	return strings.Join(parts, "+")
}

// legends are what the physical keys print, drawn as keycaps rather than
// spelled out; the side is in Name, shown where the key is chosen.
var legends = map[runtime.HoldKey]string{
	runtime.HoldFn:         "fn",
	runtime.HoldLeftOption: "⌥", runtime.HoldLeftCommand: "⌘", runtime.HoldLeftCtrl: "Ctrl", runtime.HoldLeftAlt: "Alt",
	runtime.HoldLeftShift: "Shift", runtime.HoldRightShift: "Shift", runtime.HoldMenu: "Menu",
	runtime.HoldRightOption:  "⌥",
	runtime.HoldRightCommand: "⌘",
	runtime.HoldRightAlt:     "Alt",
	runtime.HoldRightCtrl:    "Ctrl",
}

// Legend is the key's keycap legend.
func Legend(key runtime.HoldKey) string {
	return legend(key, platform.GOOS)
}

func legend(key runtime.HoldKey, os string) string {
	if legend, ok := legends[key]; ok {
		return legend
	}
	alt, super := modifierNames(os)
	parts := strings.Split(string(key), "+")
	for i, part := range parts {
		switch part {
		case "alt":
			parts[i] = alt
		case "super":
			parts[i] = super
		default:
			if legend, ok := punctuation[runtime.HoldKey(part)]; ok {
				parts[i] = legend
			} else {
				parts[i] = strings.ToUpper(part)
			}
		}
	}
	return strings.Join(parts, "+")
}

// SuperName follows the physical key name of the current platform.
func SuperName() string {
	_, super := modifierNames(platform.GOOS)
	return super
}

// AltName follows the native Option/Alt key legend.
func AltName() string {
	alt, _ := modifierNames(platform.GOOS)
	return alt
}

func modifierNames(os string) (alt, super string) {
	if os == "darwin" {
		return "Option", "Command"
	}
	return "Alt", "Win"
}
