//go:build !js

package mygo

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
)

// untrustedWebView returns a web view whose page is untrusted, though it
// trusts every origin, with a page of the custom scheme "plugin" loaded
// and ready.
func untrustedWebView(t *testing.T) (*Window, *WebView, *fake.WebView) {
	t.Helper()
	if err := Protocol.HandleFunc("plugin", func(w http.ResponseWriter, r *http.Request) {
		p := CallerPage(r.Context())
		if p == nil || p.WebView() == nil {
			http.Error(w, "no page", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "web view %d in window %d", p.WebView().w.id, CallerWindow(r.Context()).ID())
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Protocol.Unhandle("plugin") })
	show := true
	w, v, _, fv := webViewWindowOpts(t, &show, WebViewOptions{Page: PageOptions{Untrusted: true, TrustedOrigins: []string{"*"}}})
	onMain(func() { fv.H.NavigationCommitted("plugin://localhost/index.html") })
	page(fv.Window, `{"t":"dom-ready"}`)
	return w, v, fv
}

// An untrusted page calls no Go method, bound or MyGo's own, whatever its
// URL and the origins its web view trusts.
func TestUntrustedCalls(t *testing.T) {
	bindForTest(t, "Pages", pageService{})
	w, _, fv := untrustedWebView(t)
	for i, method := range []string{"Pages.Where", "mygo:window.SetTitle", "mygo:window.Close"} {
		m := call(t, fv.Window, i+1, method, "x")
		if m["ok"] != false || !strings.Contains(fmt.Sprint(m["e"]), "untrusted") {
			t.Errorf("%s from an untrusted page: %v", method, m)
		}
	}
	if w.IsDestroyed() {
		t.Error("an untrusted page closed its window")
	}
	// Messages above the limit are dropped before they are decoded: a
	// call that fits gets its answer, one that does not gets none.
	big := strings.Repeat("a", maxUntrustedMessage)
	page(fv.Window, fmt.Sprintf(`{"t":"call","id":10,"k":"tok","m":"Pages.Where","a":[%q]}`, big))
	call(t, fv.Window, 11, "Pages.Where")
	for _, s := range fv.Scripts() {
		if strings.Contains(s, `"id":10`) {
			t.Errorf("a message above the limit was decoded: %.200s", s)
		}
	}
}

// Untrusted pages hear only the events sent to them.
func TestUntrustedEvents(t *testing.T) {
	ev := newEventForTest[string](t, "untrusted-test")
	w, v, fv := untrustedWebView(t)
	if err := ev.Broadcast("to all"); err != nil {
		t.Fatal(err)
	}
	if err := ev.Emit(w, "to the window"); err != nil {
		t.Fatal(err)
	}
	if err := ev.EmitPage(v.Page(), "to the page"); err != nil {
		t.Fatal(err)
	}
	received(t, fv.Window, func(m map[string]any) bool { return m["t"] == "event" && m["p"] == "to the page" })
	for _, s := range fv.Scripts() {
		if strings.Contains(s, "to all") || strings.Contains(s, "to the window") {
			t.Errorf("an untrusted page heard %s", s)
		}
	}
}

// Without handlers of the app's, an untrusted page gets no permission,
// opens no window, downloads nothing and navigates only within its
// origin; the handlers decide when there are.
func TestUntrustedDefaults(t *testing.T) {
	dir := t.TempDir()
	App.SetPath(PathDownloads, dir)
	defer App.SetPath(PathDownloads, "")
	_, v, fv := untrustedWebView(t)
	p := v.Page()
	h := fv.H

	if onMainValue(func() bool { return h.PermissionRequested([]string{"camera"}, "plugin://localhost") }) {
		t.Error("an untrusted page got the camera without a permission handler")
	}
	p.SetPermissionHandler(func(PermissionRequest) bool { return true })
	if !onMainValue(func() bool { return h.PermissionRequested([]string{"camera"}, "plugin://localhost") }) {
		t.Error("the permission handler of an untrusted page was not asked")
	}

	// The page of a window, which opens its links in the browser, tells
	// when the untrusted page's would have opened too.
	opened := 0
	off := App.OnWindowCreated(func(c *Window) { opened++; t.Cleanup(c.Destroy) })
	defer off()
	_, fw := testWindow(t, WindowOptions{})
	opened = 0
	onMain(func() { h.NewWindow(platform.NewWindowRequest{URL: "https://untrusted.example/", UserInitiated: true}) })
	onMain(func() {
		fw.H.NewWindow(platform.NewWindowRequest{URL: "https://trusted.example/", UserInitiated: true})
	})
	wait.Until(t, "the browser to open", func() bool {
		return onMainValue(func() bool { return slices.Contains(fake.OpenedExternal, "https://trusted.example/") })
	})
	if opened != 0 || onMainValue(func() bool { return slices.Contains(fake.OpenedExternal, "https://untrusted.example/") }) {
		t.Error("an untrusted page opened a window or the browser without a handler")
	}

	start := func() string {
		return onMainValue(func() string { return h.DownloadStarted("plugin://localhost/a", "a.txt") })
	}
	if got := start(); got != "" {
		t.Errorf("an untrusted page downloads to %q without listeners", got)
	}
	p.OnWillDownload(func(e *DownloadEvent) {})
	if got := start(); got != filepath.Join(dir, "a.txt") {
		t.Errorf("with a listener, the download goes to %q", got)
	}
	os.Remove(filepath.Join(dir, "a.txt"))

	navigates := func(url string, unasked bool) bool {
		return onMainValue(func() bool {
			return h.WillNavigate(platform.Navigation{URL: url, IsMainFrame: true, UserInitiated: true, Unasked: unasked})
		})
	}
	for url, want := range map[string]bool{
		"plugin://localhost/other.html":        true,
		"PLUGIN://LOCALHOST/#top":              true,
		"blob:plugin://localhost/1234":         true,
		"plugin://elsewhere/":                  false,
		"https://example.com/":                 false,
		"blob:https://example.com/1234":        false,
		"data:text/html,<p>somewhere else</p>": false,
	} {
		if got := navigates(url, false); got != want {
			t.Errorf("an untrusted page navigating to %s: %v, want %v", url, got, want)
		}
	}
	if !navigates("https://example.com/", true) {
		t.Error("the app's own load was refused")
	}
	if !onMainValue(func() bool {
		return h.WillNavigate(platform.Navigation{URL: "https://example.com/", UserInitiated: true})
	}) {
		t.Error("a frame's navigation was refused")
	}
	var asked []string
	p.OnWillNavigate(func(e *NavigateEvent) { asked = append(asked, e.URL) })
	if !navigates("https://example.com/", false) || len(asked) != 1 {
		t.Errorf("with a listener: asked %v", asked)
	}
}

// The handlers of custom schemes tell which page asks, and untrusted
// pages get no CORS headers for other origins.
func TestUntrustedSchemeRequests(t *testing.T) {
	w, v, fv := untrustedWebView(t)
	rec := request(t, fv.Window, "POST", "plugin://localhost/msg", strings.NewReader("hi"))
	if want := fmt.Sprintf("web view %d in window %d", v.w.id, w.ID()); rec.body.String() != want {
		t.Errorf("the handler got %q, want %q", rec.body.String(), want)
	}
	rec = &recorder{finished: make(chan struct{})}
	req := &platform.SchemeRequest{Context: t.Context(), Method: "GET", URL: "plugin://localhost/", Header: http.Header{"Origin": {"https://example.com"}}, Responder: rec}
	onMain(func() { fv.H.SchemeRequest(req) })
	<-rec.finished
	if got := rec.header["Access-Control-Allow-Origin"]; got != nil {
		t.Errorf("an untrusted page got CORS headers %v", got)
	}
}
