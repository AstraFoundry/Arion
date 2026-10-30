package bootstrap

import (
	"arion.local/core/domain"
	arion "arion.local/desktop/ui/shared/runtime"
	"bytes"
	"github.com/egoist/mygo"
	"image"
	"image/png"
	"testing"
	"time"
)

func TestClipboardHoverBitmapIsBoundedAndValidated(t *testing.T) {
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1280, 640))); err != nil {
		t.Fatal(err)
	}
	bitmap, err := clipBitmap(data.Bytes(), 640)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := bitmap.Size(); w != 640 || h != 320 {
		t.Fatalf("preview dimensions = %dx%d", w, h)
	}
	if _, err := clipBitmap([]byte("invalid image"), 640); err == nil {
		t.Fatal("invalid preview image accepted")
	}
}

func TestClipboardHoverDismissalDropsInFlightImage(t *testing.T) {
	a := clipboardFixture(t)
	payload := domain.ClipPayload{Kind: domain.ClipImage, PNG: []byte("invalid image")}
	item := domain.Clip{ID: payload.ID(), Payload: payload}
	a.state.Clipboard.Items = []arion.Clip{item}
	// The window is only an identity sentinel; no native operation may run after dismissal.
	a.clipboardWin = &mygo.Window{}
	cc := a.clipboard
	cc.previewID = item.ID
	cc.previewSeq = 1
	a.prepareClipboardPreview()
	a.prepareClipboardPreview()
	if !cc.previewLoading || a.resourceJobs.Load() != 1 {
		t.Fatal("hover started duplicate image workers")
	}
	a.dismissClipboardPreview()
	a.clipboardWin = nil
	select {
	case <-a.queueReady:
		a.queueMu.Lock()
		batch := a.queue
		a.queue = nil
		a.queueMu.Unlock()
		for _, fn := range batch {
			fn()
		}
	case <-time.After(5 * time.Second):
		t.Fatal("preview worker did not complete")
	}
	if cc.previewWindow != nil || cc.previewID != "" || cc.previewLoading || a.resourceJobs.Load() != 0 || a.state.Clipboard.Error != "" {
		t.Fatal("dismissed preview retained work or published stale failure")
	}
}
