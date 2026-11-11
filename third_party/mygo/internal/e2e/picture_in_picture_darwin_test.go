//go:build darwin

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
)

// allowsPictureInPicture reports whether a window's page lets its videos
// go into picture in picture.
func allowsPictureInPicture(w *mygo.Window) (allowed, supported bool) {
	mygo.RunOnMain(func() { allowed, supported = darwin.TestAllowsPictureInPicture(w.NativeHandle()) })
	return allowed, supported
}

// PictureInPicture turns WebKit's preference on, which WKWebView keeps
// off outside Safari.
func TestPictureInPicturePreference(t *testing.T) {
	plain := newWindow(t, mygo.WindowOptions{Title: "No picture in picture", Width: 300, Height: 200})
	allowed, supported := allowsPictureInPicture(plain)
	if !supported {
		t.Skip("WebKit doesn't answer _allowsPictureInPictureMediaPlayback")
	}
	if allowed {
		t.Error("a page without PictureInPicture lets its videos go into picture in picture")
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Picture in picture", Width: 300, Height: 200, Page: mygo.PageOptions{PictureInPicture: true}})
	if allowed, _ := allowsPictureInPicture(w); !allowed {
		t.Error("PictureInPicture did not turn WebKit's preference on")
	}
}
