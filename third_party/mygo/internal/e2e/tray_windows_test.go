//go:build windows

package e2e

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo"
)

// TestTrayAdded shows tray icons numbered by their IDs once NewTray is
// done, with or without an image, and where they are as soon as asked,
// each again once made again.
func TestTrayAdded(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	img.Set(0, 0, color.Black)
	var icon bytes.Buffer
	if err := png.Encode(&icon, img); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		withIcon, err := mygo.NewTray(mygo.TrayOptions{ID: "e2e.icon", Icon: icon.Bytes(), ToolTip: "e2e icon"})
		if err != nil {
			t.Fatal(err)
		}
		if b := withIcon.Bounds(); b.Width == 0 || b.Height == 0 {
			t.Errorf("an icon made with an image is not in the notification area: %+v", b)
		}
		blank, err := mygo.NewTray(mygo.TrayOptions{ID: "e2e.blank"})
		if err != nil {
			t.Fatal(err)
		}
		eventually(t, "the blank icon added", func() bool { b := blank.Bounds(); return b.Width > 0 && b.Height > 0 })
		withIcon.Destroy()
		blank.Destroy()
	}
}
