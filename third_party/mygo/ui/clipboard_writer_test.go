package ui

import "testing"

func TestContentClipboardWriterIsScopedAndPreservesText(t *testing.T) {
	base := View(func(*Context) {})
	var copied string
	custom := base.WithClipboardWriter(func(s string) { copied = s })
	if base.clipboardWrite != nil || custom == base || custom.clipboardWrite == nil {
		t.Fatal("writer mutated shared content")
	}
	// With no native connection, reaching the default clipboard would panic.
	h := &windowHost{clipboardWrite: custom.clipboardWrite}
	h.writeClipboard("selected 中文🙂\nsecond line")
	if copied != "selected 中文🙂\nsecond line" {
		t.Fatal("copy changed text")
	}
	reset := custom.WithClipboardWriter(nil)
	if reset.clipboardWrite != nil || custom.clipboardWrite == nil {
		t.Fatal("writer reset escaped its content")
	}
}
