//go:build darwin

package e2e

import (
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// TestDialogFromClick shows each kind of dialog from a button's Clicked,
// as the frame that reports the click runs: the dialog shows and waits
// until it is dismissed. A frame drawn as AppKit displayed the view ran in
// its display cycle, where the panel never showed and runModal returned at
// once, and a sheet never showed nor ended (#883). The window draws again
// once the dialog is gone.
func TestDialogFromClick(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		show func(parent *mygo.Window)
	}{
		{"open", func(*mygo.Window) {
			mygo.Dialog.Open(mygo.OpenDialogOptions{DefaultPath: dir, Directory: true, Multiple: true})
		}},
		{"save", func(*mygo.Window) {
			mygo.Dialog.Save(mygo.SaveDialogOptions{DefaultPath: filepath.Join(dir, "note.txt")})
		}},
		{"message", func(*mygo.Window) {
			mygo.Dialog.Message(mygo.MessageOptions{Message: "From a click", Buttons: []string{"OK", "Cancel"}})
		}},
		{"open sheet", func(w *mygo.Window) {
			mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: w, DefaultPath: dir})
		}},
		{"message sheet", func(w *mygo.Window) {
			mygo.Dialog.Message(mygo.MessageOptions{Parent: w, Message: "From a click"})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w *mygo.Window // main thread only
			var frames atomic.Int32
			var clicked, returned atomic.Bool
			view := func(c *ui.Context) {
				frames.Add(1)
				if ui.Button(c, "Show").Width(120).Height(30).Clicked() {
					clicked.Store(true)
					tc.show(w)
					returned.Store(true)
				}
			}
			var handle uintptr
			mygo.RunOnMain(func() {
				w = mygo.NewWindow(mygo.WindowOptions{Title: "Dialog from a click", Width: 300, Height: 200, Content: ui.View(view)})
				handle = w.NativeHandle()
			})
			win := w
			t.Cleanup(win.Destroy)
			eventually(t, "a frame", func() bool { return frames.Load() > 0 })
			// Frames pause, so that the click's frame is one after a pause,
			// which AppKit drew as it displayed the view.
			time.Sleep(200 * time.Millisecond)
			// The button lays out at the top-left of the window.
			click(win, 60, 15)
			eventually(t, "the click", clicked.Load)
			eventually(t, "the dialog to show", func() bool {
				var running, shows bool
				mygo.RunOnMain(func() { running, shows = darwin.TestModal(handle) })
				return running && shows
			})
			if returned.Load() {
				t.Fatal("the dialog returned before it was dismissed")
			}
			var canceled bool
			mygo.RunOnMain(func() { canceled = darwin.TestCancelModal(handle) })
			if !canceled {
				t.Fatal("no dialog to dismiss")
			}
			eventually(t, "the dialog to return", returned.Load)
			n := frames.Load()
			win.Invalidate()
			eventually(t, "a frame after the dialog", func() bool { return frames.Load() > n })
		})
	}
}

func TestMessageButtonRoles(t *testing.T) {
	type key struct {
		code  uint16
		chars string
		cmd   bool
	}
	ret, esc, cmdPeriod := key{36, "\r", false}, key{53, "\x1b", false}, key{47, ".", true}
	cancelIsDefault := mygo.MessageOptions{Buttons: []string{"Continue Editing", "Discard"}, DestructiveButtons: []int{1}}
	// The first button must not be destructive: AppKit may suppress Return
	// for it even when it is the default.
	noDefault := mygo.MessageOptions{Buttons: []string{"Continue", "Cancel"}, DefaultButton: -1}
	cases := []struct {
		name        string
		opts        mygo.MessageOptions
		keys        []key
		destructive []bool
		want        int
	}{
		{"Return on a default that cancels", cancelIsDefault, []key{ret}, []bool{false, true}, 0},
		{"Escape on a default that cancels", cancelIsDefault, []key{esc}, []bool{false, true}, 0},
		{"Return and Cmd+. without a default", noDefault, []key{ret, cmdPeriod}, []bool{false, false}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w *mygo.Window
			mygo.RunOnMain(func() {
				w = mygo.NewWindow(mygo.WindowOptions{Title: "Message keys", Width: 300, Height: 200, Content: ui.View(func(*ui.Context) {})})
			})
			t.Cleanup(w.Destroy)
			eventually(t, "the window to show", w.IsVisible)
			answered := make(chan mygo.MessageResult, 1)
			go func() {
				opts := tc.opts
				opts.Parent, opts.Message = w, "Message keys"
				res, err := mygo.Dialog.Message(opts)
				if err != nil {
					t.Error(err)
				}
				answered <- res
			}()
			handle := w.NativeHandle()
			var destructive []bool
			eventually(t, "the sheet", func() bool {
				var ok bool
				mygo.RunOnMain(func() { destructive, ok = darwin.TestSheetDestructive(handle) })
				return ok && len(destructive) == len(tc.destructive)
			})
			if !slices.Equal(destructive, tc.destructive) {
				t.Errorf("destructive buttons %v, want %v", destructive, tc.destructive)
			}
			for _, k := range tc.keys {
				mygo.RunOnMain(func() { darwin.TestSheetKey(handle, k.code, k.chars, k.cmd) })
			}
			if res := wait.For(t, "the message's answer", answered); res.Button != tc.want {
				t.Errorf("the keys answered button %d, want %d", res.Button, tc.want)
			}
		})
	}
}
