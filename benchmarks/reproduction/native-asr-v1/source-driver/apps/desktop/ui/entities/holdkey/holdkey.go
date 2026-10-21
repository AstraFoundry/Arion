// Package holdkey names the dictation key, the only global activation.
package holdkey

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

var names = map[runtime.HoldKey]copy.Key{
	runtime.HoldFn:           copy.HoldKeyFn,
	runtime.HoldRightOption:  copy.HoldKeyRightOption,
	runtime.HoldRightCommand: copy.HoldKeyRightCommand,
	runtime.HoldRightAlt:     copy.HoldKeyRightAlt,
	runtime.HoldRightCtrl:    copy.HoldKeyRightCtrl,
}

// Name is the key's display name.
func Name(messages copy.Catalog, key runtime.HoldKey) string { return messages.Text(names[key]) }

// legends are what the physical keys print, drawn as keycaps rather than
// spelled out; the side is in Name, shown where the key is chosen.
var legends = map[runtime.HoldKey]string{
	runtime.HoldFn:           "fn",
	runtime.HoldRightOption:  "⌥",
	runtime.HoldRightCommand: "⌘",
	runtime.HoldRightAlt:     "Alt",
	runtime.HoldRightCtrl:    "Ctrl",
}

// Legend is the key's keycap legend.
func Legend(key runtime.HoldKey) string { return legends[key] }
