//go:build !js

package mygo

import (
	"errors"
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

// TestPreloadAllFrames: the preload script runs in the top frame only,
// unless PreloadAllFrames asks for the iframes too.
func TestPreloadAllFrames(t *testing.T) {
	for _, all := range []bool{false, true} {
		w := NewWindow(WindowOptions{Hidden: true, URL: "https://example.com/", Page: PageOptions{PreloadScript: "window.sniff = 1", PreloadAllFrames: all}})
		fw := fb.Windows()[len(fb.Windows())-1]
		i := slices.IndexFunc(fw.Opts.UserScripts, func(s platform.UserScript) bool { return s.Source == "window.sniff = 1" })
		if i < 0 || fw.Opts.UserScripts[i].AllFrames != all {
			t.Errorf("PreloadAllFrames %v: scripts %+v", all, fw.Opts.UserScripts)
		}
		for _, s := range fw.Opts.UserScripts[:max(i, 0)] {
			if s.AllFrames {
				t.Errorf("PreloadAllFrames %v: the bridge runs in iframes", all)
			}
		}
		w.Destroy()
	}
}

func TestPage(t *testing.T) {
	w := NewWindow(WindowOptions{Hidden: true, URL: "https://example.com/a", Page: PageOptions{
		PreloadScript: "window.preloaded = true", UserAgent: "agent", UserAgentSuffix: " MyApp/1.2 ", ZoomFactor: 1.5, DevTools: DevToolsEnabled,
	}})
	t.Cleanup(w.Destroy)
	fw := fb.Windows()[len(fb.Windows())-1]
	p := w.Page()
	if p == nil || p.Window() != w {
		t.Fatalf("Page %v", p)
	}
	o := fw.Opts
	preloaded := slices.ContainsFunc(o.UserScripts, func(s platform.UserScript) bool { return s.Source == "window.preloaded = true" })
	if o.UserAgent != "agent" || o.UserAgentSuffix != "MyApp/1.2" || o.Zoom != 1.5 || !o.DevTools || !preloaded {
		t.Errorf("page options: user agent %q, suffix %q, zoom %v, devtools %v, preloaded %v", o.UserAgent, o.UserAgentSuffix, o.Zoom, o.DevTools, preloaded)
	}
	if got := p.URL(); got != "https://example.com/a" {
		t.Errorf("URL %q", got)
	}
	if err := p.LoadURL("https://example.com/b"); err != nil || p.URL() != "https://example.com/b" {
		t.Errorf("LoadURL: %v, URL %q", err, p.URL())
	}
	if got := p.ProcessID(); got != 0 {
		t.Errorf("ProcessID before the backend knows one: %d", got)
	}
	fw.SetProcessID(4242)
	if got := p.ProcessID(); got != 4242 {
		t.Errorf("ProcessID %d, want 4242", got)
	}
	if got := p.GPUProcessID(); got != 0 {
		t.Errorf("GPUProcessID before the backend knows one: %d", got)
	}
	fw.SetGPUProcessID(4343)
	if got := p.GPUProcessID(); got != 4343 {
		t.Errorf("GPUProcessID %d, want 4343", got)
	}
	if _, err := EvalAs[int]((*Page)(nil), "1"); !errors.Is(err, errNoPage) {
		t.Errorf("EvalAs of no page: %v", err)
	}
}

func TestPageProcessIDAfterClose(t *testing.T) {
	w := NewWindow(WindowOptions{Hidden: true, URL: "https://example.com/"})
	fb.Windows()[len(fb.Windows())-1].SetProcessID(7)
	fb.Windows()[len(fb.Windows())-1].SetGPUProcessID(8)
	p := w.Page()
	if got := p.ProcessID(); got != 7 {
		t.Fatalf("ProcessID %d, want 7", got)
	}
	if got := p.GPUProcessID(); got != 8 {
		t.Fatalf("GPUProcessID %d, want 8", got)
	}
	w.Destroy()
	if got := p.ProcessID(); got != 0 {
		t.Errorf("ProcessID of a closed window's page: %d", got)
	}
	if got := p.GPUProcessID(); got != 0 {
		t.Errorf("GPUProcessID of a closed window's page: %d", got)
	}
}

func TestPageThemeColor(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	p := w.Page()
	type change struct {
		c  color.NRGBA
		ok bool
	}
	var changes []change // listeners run on the main thread, as page does
	off := p.OnThemeColorChanged(func(c color.NRGBA, ok bool) { changes = append(changes, change{c, ok}) })
	defer off()
	expect := func(what string, want ...change) {
		t.Helper()
		if !slices.Equal(changes, want) {
			t.Errorf("%s: changes %v, want %v", what, changes, want)
		}
		changes = nil
	}
	if c, ok := p.ThemeColor(); ok {
		t.Errorf("ThemeColor of a new page: %v", c)
	}

	blue := color.NRGBA{R: 0x1e, G: 0x90, B: 0xff, A: 0xff}
	page(fw, `{"t":"theme-color","v":"#1e90ffff"}`)
	if c, ok := p.ThemeColor(); !ok || c != blue {
		t.Errorf("ThemeColor = %v, %v; want %v", c, ok, blue)
	}
	expect("a meta", change{blue, true})
	page(fw, `{"t":"theme-color","v":"#1e90ffff"}`)
	expect("the same color")
	half := color.NRGBA{R: 255, A: 0x80}
	page(fw, `{"t":"theme-color","v":"#ff000080"}`)
	expect("a translucent color", change{half, true})
	page(fw, `{"t":"theme-color","v":""}`)
	if c, ok := p.ThemeColor(); ok {
		t.Errorf("ThemeColor after the meta went: %v", c)
	}
	expect("the meta removed", change{})
	page(fw, `{"t":"theme-color","v":""}`)
	page(fw, `{"t":"theme-color","v":"not a color"}`)
	expect("no color still")

	// Another page keeps the color until it tells its own, and changes
	// nothing when that is the same.
	page(fw, `{"t":"theme-color","v":"#1e90ffff"}`)
	expect("a meta again", change{blue, true})
	onMain(func() { fw.H.NavigationCommitted("https://next.example/") })
	if c, ok := p.ThemeColor(); !ok || c != blue {
		t.Errorf("ThemeColor as the next page loads = %v, %v", c, ok)
	}
	page(fw, `{"t":"theme-color","v":"#1e90ffff"}`)
	page(fw, `{"t":"dom-ready"}`)
	onMain(fw.H.LoadFinished)
	expect("a page of the same color")

	// A page without the bridge, an image, has none once it loaded.
	onMain(func() { fw.H.NavigationCommitted("https://next.example/a.png") })
	expect("an image loading")
	onMain(fw.H.LoadFinished)
	expect("an image loaded", change{})
	onMain(func() { fw.H.LoadFailed("https://next.example/b", 1, "nope") })
	expect("a failed load")

	off()
	page(fw, `{"t":"theme-color","v":"#1e90ffff"}`)
	expect("after off")
	w.Destroy()
	if c, ok := p.ThemeColor(); ok {
		t.Errorf("ThemeColor of a closed window's page: %v", c)
	}
}

// Only the page's bridge tells the theme color, not another frame, which
// lacks its secret.
func TestPageThemeColorNeedsSecret(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	onMain(func() { fw.H.Message(`{"t":"theme-color","v":"#ff0000ff"}`) })
	if c, ok := w.Page().ThemeColor(); ok {
		t.Errorf("ThemeColor from a message without the secret: %v", c)
	}
}

func TestPageAudio(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	p := w.Page()
	var changes []bool // listeners run on the main thread, as SetAudible does
	off := p.OnAudibleChanged(func(audible bool) { changes = append(changes, audible) })
	expect := func(what string, audible bool, want ...bool) {
		t.Helper()
		if got := p.IsAudible(); got != audible {
			t.Errorf("%s: IsAudible %v, want %v", what, got, audible)
		}
		if !slices.Equal(changes, want) {
			t.Errorf("%s: changes %v, want %v", what, changes, want)
		}
		changes = nil
	}
	expect("a new page", false)
	onMain(func() { fw.SetAudible(true) })
	expect("playing", true, true)
	onMain(func() { fw.SetAudible(true) })
	expect("playing again", true) // told only when it changes
	onMain(func() { fw.SetAudible(false) })
	expect("stopped", false, false)
	off()
	onMain(func() { fw.SetAudible(true) })
	expect("after off", true)

	if p.IsAudioMuted() {
		t.Error("a new page is muted")
	}
	p.SetAudioMuted(true)
	if !p.IsAudioMuted() || !fw.IsAudioMuted() {
		t.Errorf("SetAudioMuted(true): IsAudioMuted %v, backend's %v", p.IsAudioMuted(), fw.IsAudioMuted())
	}
	p.SetAudioMuted(false)
	if p.IsAudioMuted() || fw.IsAudioMuted() {
		t.Errorf("SetAudioMuted(false): IsAudioMuted %v, backend's %v", p.IsAudioMuted(), fw.IsAudioMuted())
	}

	p.SetAudioMuted(true)
	w.Destroy()
	if p.IsAudible() || p.IsAudioMuted() {
		t.Errorf("a closed window's page: IsAudible %v, IsAudioMuted %v", p.IsAudible(), p.IsAudioMuted())
	}
	p.SetAudioMuted(false) // does nothing
}

// TestPageBackgroundThrottling passes BackgroundThrottling to the platform,
// for a window's page and a web view's: ThrottlingOn, the zero value,
// leaves the page to the web view.
func TestPageBackgroundThrottling(t *testing.T) {
	for _, tt := range []struct {
		throttling BackgroundThrottling
		off        bool
	}{{ThrottlingOn, false}, {ThrottlingOff, true}} {
		w := NewWindow(WindowOptions{Hidden: true, Page: PageOptions{BackgroundThrottling: tt.throttling}})
		fw := fb.Windows()[len(fb.Windows())-1]
		if got := fw.Opts.NoBackgroundThrottling; got != tt.off {
			t.Errorf("window with throttling %d: NoBackgroundThrottling %v, want %v", tt.throttling, got, tt.off)
		}
		w.Destroy()
	}
	w, _, s := contentWindow(t, func(c *ui.Context) {})
	for _, throttling := range []BackgroundThrottling{ThrottlingOff, ThrottlingOn} {
		if _, err := w.NewWebView(WebViewOptions{Page: PageOptions{BackgroundThrottling: throttling}}); err != nil {
			t.Fatal(err)
		}
	}
	if views := s.WebViews(); len(views) != 2 || !views[0].Opts.NoBackgroundThrottling || views[1].Opts.NoBackgroundThrottling {
		t.Errorf("web views %d: BackgroundThrottling not applied", len(views))
	}
}

// TestPageAutoplay passes Autoplay to the platform, for a window's page and
// a web view's: AutoplayUserGesture, the zero value, leaves the page to
// the web view's policy.
func TestPageAutoplay(t *testing.T) {
	for _, tt := range []struct {
		autoplay Autoplay
		allow    bool
	}{{AutoplayUserGesture, false}, {AutoplayAllow, true}} {
		w := NewWindow(WindowOptions{Hidden: true, Page: PageOptions{Autoplay: tt.autoplay}})
		fw := fb.Windows()[len(fb.Windows())-1]
		if got := fw.Opts.Autoplay; got != tt.allow {
			t.Errorf("window with autoplay %d: Autoplay %v, want %v", tt.autoplay, got, tt.allow)
		}
		w.Destroy()
	}
	w, _, s := contentWindow(t, func(c *ui.Context) {})
	for _, autoplay := range []Autoplay{AutoplayAllow, AutoplayUserGesture} {
		if _, err := w.NewWebView(WebViewOptions{Page: PageOptions{Autoplay: autoplay}}); err != nil {
			t.Fatal(err)
		}
	}
	if views := s.WebViews(); len(views) != 2 || !views[0].Opts.Autoplay || views[1].Opts.Autoplay {
		t.Errorf("web views %d: Autoplay not applied", len(views))
	}
}

func TestPageCapture(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true})
	p := w.Page()
	type change struct {
		kind  CaptureKind
		state CaptureState
	}
	var changes []change // listeners run on the main thread, as SetCapture does
	off := p.OnCaptureStateChanged(func(kind CaptureKind, state CaptureState) {
		changes = append(changes, change{kind, state})
	})
	expect := func(what string, camera, microphone, display CaptureState, want ...change) {
		t.Helper()
		for kind, s := range map[CaptureKind]CaptureState{CaptureCamera: camera, CaptureMicrophone: microphone, CaptureDisplay: display} {
			if got := p.CaptureState(kind); got != s {
				t.Errorf("%s: CaptureState(%d) %d, want %d", what, kind, got, s)
			}
		}
		if !slices.Equal(changes, want) {
			t.Errorf("%s: changes %v, want %v", what, changes, want)
		}
		changes = nil
	}
	expect("a new page", CaptureNone, CaptureNone, CaptureNone)

	// The page muted nothing: setting does nothing either.
	p.SetCaptureState(CaptureMicrophone, CaptureMuted)
	expect("muting a microphone the page doesn't capture", CaptureNone, CaptureNone, CaptureNone)

	onMain(func() {
		fw.SetCapture(platform.CaptureCamera, platform.CaptureActive)
		fw.SetCapture(platform.CaptureMicrophone, platform.CaptureActive)
	})
	expect("a call", CaptureActive, CaptureActive, CaptureNone,
		change{CaptureCamera, CaptureActive}, change{CaptureMicrophone, CaptureActive})
	onMain(func() { fw.SetCapture(platform.CaptureCamera, platform.CaptureActive) })
	expect("the camera again", CaptureActive, CaptureActive, CaptureNone) // told only when it changes

	p.SetCaptureState(CaptureMicrophone, CaptureMuted)
	expect("the microphone muted", CaptureActive, CaptureMuted, CaptureNone, change{CaptureMicrophone, CaptureMuted})
	p.SetCaptureState(CaptureMicrophone, CaptureActive)
	expect("the microphone unmuted", CaptureActive, CaptureActive, CaptureNone, change{CaptureMicrophone, CaptureActive})
	p.SetCaptureState(CaptureCamera, CaptureNone)
	expect("the camera stopped", CaptureNone, CaptureActive, CaptureNone, change{CaptureCamera, CaptureNone})
	onMain(func() { fw.SetCapture(platform.CaptureDisplay, platform.CaptureActive) })
	expect("sharing the screen", CaptureNone, CaptureActive, CaptureActive, change{CaptureDisplay, CaptureActive})

	// Unknown kinds and states.
	p.SetCaptureState(CaptureKind(3), CaptureMuted)
	p.SetCaptureState(CaptureKind(-1), CaptureMuted)
	p.SetCaptureState(CaptureMicrophone, CaptureState(3))
	p.SetCaptureState(CaptureMicrophone, CaptureState(-1))
	if got := p.CaptureState(CaptureKind(3)); got != CaptureNone {
		t.Errorf("CaptureState of an unknown kind: %d", got)
	}
	onMain(func() { fw.H.CaptureStateChanged(platform.CaptureKind(3), platform.CaptureActive) })
	expect("unknown kinds and states", CaptureNone, CaptureActive, CaptureActive)

	off()
	onMain(func() { fw.SetCapture(platform.CaptureDisplay, platform.CaptureNone) })
	expect("after off", CaptureNone, CaptureActive, CaptureNone)

	w.Destroy()
	if got := p.CaptureState(CaptureMicrophone); got != CaptureNone {
		t.Errorf("a closed window's page: CaptureState %d", got)
	}
	p.SetCaptureState(CaptureMicrophone, CaptureMuted) // does nothing
}

// TestPagePictureInPicture passes PictureInPicture to the backend, and
// moves the page's video into picture in picture and back.
func TestPagePictureInPicture(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Hidden: true, Page: PageOptions{PictureInPicture: true}})
	if !fw.Opts.PictureInPicture {
		t.Error("PictureInPicture not passed to the backend")
	}
	if _, plain := testWindow(t, WindowOptions{Hidden: true}); plain.Opts.PictureInPicture {
		t.Error("picture in picture on by default")
	}
	p := w.Page()
	if p.IsPictureInPictureActive() {
		t.Error("a new page's video in picture in picture")
	}
	p.TogglePictureInPicture()
	if !p.IsPictureInPictureActive() {
		t.Error("TogglePictureInPicture did not move the video out")
	}
	// From the main thread, as an event listener asks.
	if !onMainValue(p.IsPictureInPictureActive) {
		t.Error("IsPictureInPictureActive on the main thread: false")
	}
	p.TogglePictureInPicture()
	if p.IsPictureInPictureActive() {
		t.Error("TogglePictureInPicture again did not move the video back")
	}

	p.TogglePictureInPicture()
	w.Destroy()
	if p.IsPictureInPictureActive() {
		t.Error("a closed window's page: IsPictureInPictureActive true")
	}
	p.TogglePictureInPicture() // does nothing
}
