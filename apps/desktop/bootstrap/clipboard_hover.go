package bootstrap

import (
	"arion.local/core/domain"
	uiapp "arion.local/desktop/ui/app"
	"arion.local/desktop/ui/shared/copy"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// HoverClipboard receives only a settled content hover, never keyboard selection.
func (x actions) HoverClipboard(id string, rect ui.Rect, height int) {
	a, cc := x.a, x.a.clipboard
	if cc == nil {
		return
	}
	anchor := mygo.Rectangle{X: int(rect.X), Y: int(rect.Y), Width: int(rect.W), Height: int(rect.H)}
	if id == "" || a.clipboardWin == nil || a.state.Clipboard.Using {
		id, anchor, height = "", mygo.Rectangle{}, 0
	}
	if cc.previewID == id && cc.previewAnchor == anchor && cc.previewHeight == height {
		return
	}
	cc.previewID, cc.previewAnchor, cc.previewHeight = id, anchor, height
	cc.previewSeq++
	seq := cc.previewSeq
	// Native windows are changed after the parent's frame has finished building.
	a.post(func() {
		if cc.previewSeq != seq {
			return
		}
		a.closeClipboardPreviewWindow()
		a.prepareClipboardPreview()
	})
}

func (a *App) dismissClipboardPreview() {
	cc := a.clipboard
	cc.previewID, cc.previewAnchor, cc.previewHeight = "", mygo.Rectangle{}, 0
	cc.previewSeq++
	a.closeClipboardPreviewWindow()
}

func (a *App) closeClipboardPreviewWindow() {
	if window := a.clipboard.previewWindow; window != nil {
		a.clipboard.previewWindow = nil
		window.Close()
	}
}

func (a *App) prepareClipboardPreview() {
	cc := a.clipboard
	if a.clipboardWin == nil || cc.previewID == "" || cc.previewWindow != nil {
		return
	}
	item, ok := a.clipboardItem(cc.previewID)
	if !ok {
		return
	}
	if item.Payload.Kind != domain.ClipImage {
		a.showClipboardPreview(item, nil)
		return
	}
	// Decode one requested image at a time; a later hover replaces pending work.
	if cc.previewLoading {
		return
	}
	cc.previewLoading = true
	seq := cc.previewSeq
	a.resourceJobs.Add(1)
	go func() {
		bitmap, err := clipBitmap(item.Payload.PNG, 640)
		a.post(func() {
			a.resourceJobs.Add(-1)
			cc.previewLoading = false
			if seq != cc.previewSeq {
				a.prepareClipboardPreview()
				return
			}
			if a.clipboardWin == nil || cc.previewID != item.ID {
				return
			}
			if _, exists := a.clipboardItem(item.ID); !exists {
				return
			}
			if err != nil {
				a.state.Clipboard.Error = copy.ClipboardPreviewFailed
				return
			}
			a.showClipboardPreview(item, bitmap)
		})
	}()
}

func (a *App) showClipboardPreview(item domain.Clip, bitmap *ui.Bitmap) {
	cc := a.clipboard
	view := uiapp.ClipboardPreview(item, bitmap)
	window := mygo.NewFlyout(mygo.FlyoutOptions{
		Parent: a.clipboardWin, Anchor: cc.previewAnchor, Placement: mygo.PlacementRightStart, Gap: 10,
		Width: uiapp.ClipboardPreviewWidth, Height: cc.previewHeight,
		Hidden: true, Shadow: true, Content: view.Content,
	})
	cc.previewWindow = window
	configureClipboardWindow(window)
	window.SetIgnoreMouseEvents(true)
	a.track(window, view, func() {
		if cc.previewWindow == window {
			cc.previewWindow = nil
		}
	})
	window.Show()
}
