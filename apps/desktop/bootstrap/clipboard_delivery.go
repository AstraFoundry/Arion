package bootstrap

import (
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/clipboardos"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
)

func (a *App) clipboardItem(id string) (domain.Clip, bool) {
	for _, item := range a.state.Clipboard.Items {
		if item.ID == id {
			return item, true
		}
	}
	return domain.Clip{}, false
}
func (a *App) copyTranscript(text string) bool {
	unlock, ok := clipboardos.TryLock()
	if !ok {
		a.state.Clipboard.Error = copy.ClipboardWriteFailed
		return false
	}
	defer unlock()
	if _, err := clipboardos.Write(domain.ClipTextPayload(text)); err != nil {
		a.state.Clipboard.Error = copy.ClipboardWriteFailed
		return false
	}
	if a.state.Clipboard.Error == copy.ClipboardWriteFailed {
		a.state.Clipboard.Error = ""
	}
	return true
}
func (x actions) CopyClipboard(id string) {
	a := x.a
	cc := a.clipboard
	if cc == nil || cc.delivering || a.state.Clipboard.Using || a.updateApplying() {
		return
	}
	item, ok := a.clipboardItem(id)
	if !ok {
		return
	}
	seq := cc.windowSeq
	a.withClipboardResource(item, func() {
		if cc.windowSeq != seq {
			return
		}
		unlock, ok := clipboardos.TryLock()
		if !ok {
			a.state.Clipboard.Error = copy.ClipboardWriteFailed
			return
		}
		defer unlock()
		if _, err := clipboardos.Write(item.Payload); err != nil {
			a.state.Clipboard.Error = copy.ClipboardWriteFailed
			return
		}
		a.state.Clipboard.Error = ""
		x.CloseClipboard()
	})
}

func (x actions) PasteClipboard(id string) {
	if x.a.state.Clipboard.CopyOnly {
		x.CopyClipboard(id)
		return
	}

	a := x.a
	cc := a.clipboard
	if cc == nil || cc.delivering || a.state.Clipboard.Using || a.updateApplying() {
		return
	}
	item, ok := a.clipboardItem(id)
	if !ok {
		return
	}
	windowSeq := cc.windowSeq
	a.withClipboardResource(item, func() {
		if cc.windowSeq == windowSeq {
			a.pasteClipboard(item)
		}
	})
}

func (a *App) pasteClipboard(item domain.Clip) {
	cc := a.clipboard
	if cc.target == nil {
		a.state.Clipboard.Error = copy.ClipboardTargetFailed
		return
	}
	target := cc.target
	cc.target = nil
	cc.delivering = true
	a.resourceJobs.Add(1)
	cc.pasteSeq++
	seq := cc.pasteSeq
	a.state.Clipboard.Error = ""
	a.state.Clipboard.Using = true
	a.deliverClipboard(item, target, seq)
}

func (a *App) deliverClipboard(item domain.Clip, target clipboardos.Target, seq uint64) {
	cc := a.clipboard
	actions{a}.CloseClipboard()
	if err := target.Activate(); err != nil {
		target.Close()
		cc.delivering = false
		a.state.Clipboard.Using = false
		a.resourceJobs.Add(-1)
		a.state.Clipboard.Error = copy.ClipboardTargetFailed
		actions{a}.OpenClipboardSettings()
		return
	}
	deadline := time.Now().Add(700 * time.Millisecond)
	var check func()
	check = func() {
		if seq != cc.pasteSeq {
			target.Close()
			cc.delivering = false
			a.state.Clipboard.Using = false
			a.resourceJobs.Add(-1)
			return
		}
		if !target.Ready() && time.Now().Before(deadline) {
			time.AfterFunc(25*time.Millisecond, func() { a.post(check) })
			return
		}
		defer target.Close()
		defer func() {
			cc.delivering = false
			a.state.Clipboard.Using = false
			a.resourceJobs.Add(-1)
			if a.state.Clipboard.Error != "" {
				time.AfterFunc(450*time.Millisecond, func() {
					a.post(func() {
						if a.clipboardWin == nil && cc.pasteSeq == seq && a.state.Clipboard.Error != "" {
							actions{a}.OpenClipboardSettings()
						}
					})
				})
			}
		}()
		if err := target.Check(); err != nil {
			a.state.Clipboard.Error = copy.ClipboardTargetFailed
			return
		}
		unlock, ok := clipboardos.TryLock()
		if !ok {
			a.state.Clipboard.Error = copy.ClipboardWriteFailed
			return
		}
		defer unlock()
		revision, err := clipboardos.Write(item.Payload)
		if err != nil {
			a.state.Clipboard.Error = copy.ClipboardWriteFailed
			return
		}
		if err := target.Paste(revision); err != nil {
			a.state.Clipboard.Error = copy.ClipboardPasteFailed
			return
		}
		a.state.Clipboard.Error = ""
	}
	time.AfterFunc(25*time.Millisecond, func() { a.post(check) })
}

// File availability may involve a mounted filesystem. Validate off the UI
// thread and cancel actions whose history item disappeared in the meantime.
func (a *App) withClipboardResource(item domain.Clip, done func()) {
	if a.state.Clipboard.Using || a.updateApplying() {
		return
	}
	a.state.Clipboard.Using = true
	a.resourceJobs.Add(1)
	go func() {
		err := clipboardos.ValidateFiles(item.Payload)
		var meta arion.ClipboardMeta
		if item.Payload.Kind == domain.ClipFiles {
			entries, e := clipboardMetadata([]domain.Clip{item}, nil)
			if e != nil {
				err = e
			}
			meta = entries[item.ID]
		}
		a.post(func() {
			defer a.resourceJobs.Add(-1)
			a.state.Clipboard.Using = false
			if _, ok := a.clipboardItem(item.ID); !ok {
				return
			}
			if item.Payload.Kind == domain.ClipFiles {
				if a.state.Clipboard.Meta == nil {
					a.state.Clipboard.Meta = map[string]arion.ClipboardMeta{}
				}
				a.state.Clipboard.Meta[item.ID] = meta
			}
			if err != nil {
				a.state.Clipboard.Error = copy.ClipboardUnavailable
				return
			}
			done()
		})
	}()
}
