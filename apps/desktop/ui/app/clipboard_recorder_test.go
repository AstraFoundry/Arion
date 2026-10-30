package app

import (
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func (r *recorder) OpenClipboardSettings() { r.call("clipboard-settings") }

func (r *recorder) OpenClipboard()  { r.call("clipboard-open") }
func (r *recorder) CloseClipboard() { r.call("clipboard-close") }
func (r *recorder) RetryClipboard() { r.call("clipboard-retry") }
func (r *recorder) SetClipboardShortcut(s string) {
	r.state.Preferences.Clipboard.Shortcut = s
	r.call("clipboard-shortcut:" + s)
}
func (r *recorder) SetClipboardLimits(l runtime.ClipLimits) {
	r.state.Preferences.Clipboard.Limits = l
	r.call("clipboard-limits")
}
func (r *recorder) PinClipboard(id string, pin bool) { r.call("clipboard-pin:" + id) }
func (r *recorder) PasteClipboard(id string)         { r.call("clipboard-paste:" + id) }

func (r *recorder) HoverClipboard(id string, _ ui.Rect, _ int) { r.call("clipboard-hover:" + id) }

func (r *recorder) CopyClipboard(id string) { r.call("clipboard-copy:" + id) }
