//go:build !js

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// sessionServer serves a page titled "sessions" and, at /echo, the
// cookies its requests send.
func sessionServer(t *testing.T) string {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/echo" {
			fmt.Fprint(w, r.Header.Get("Cookie"))
			return
		}
		fmt.Fprint(w, "<!doctype html><title>sessions</title><body>sessions</body>")
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// sessionWindow opens url in a window of session s, nil for the default
// one, once its page loaded.
func sessionWindow(t *testing.T, s *mygo.Session, url, title string) *mygo.Window {
	t.Helper()
	w := newWindow(t, mygo.WindowOptions{Title: title, Width: 300, Height: 200, Page: mygo.PageOptions{Session: s}})
	if err := w.Page().LoadURL(url); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "document.title === 'sessions'")
	return w
}

// sessionCookie returns the value of the cookie name of 127.0.0.1 in the
// cookies of m, "" when there is none.
func sessionCookie(t *testing.T, m mygo.CookiesModule, name string) string {
	t.Helper()
	list, err := m.List(mygo.CookieFilter{Name: name, Domain: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		return c.Value
	}
	return ""
}

// TestSessions shows pages of two persistent sessions, the default one and
// an ephemeral one side by side: none sees the cookies or the local
// storage of another, their pages load custom schemes, a window a page
// opens is in the opener's session, ClearData clears what it is asked to,
// and an ephemeral session's data goes with its last page.
func TestSessions(t *testing.T) {
	url := sessionServer(t)
	name := fmt.Sprintf("s%d", time.Now().UnixNano()) // a cookie no other run set
	work, home := mygo.NewSession("e2e-work"), mygo.NewSession("e2e-home")
	if mygo.NewSession("e2e-work") != work || work.Name() != "e2e-work" || work.IsEphemeral() {
		t.Fatal("NewSession made another session of the same name")
	}
	for _, s := range []*mygo.Session{work, home} {
		if err := s.ClearData(mygo.SiteDataAll); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_ = work.ClearData(mygo.SiteDataAll)
		_ = home.ClearData(mygo.SiteDataAll)
	})

	w1 := sessionWindow(t, work, url+"/", "Work")
	w2 := sessionWindow(t, home, url+"/", "Home")
	d := sessionWindow(t, nil, url+"/", "Default")
	mustEval(t, w1, fmt.Sprintf(`(document.cookie = "%s=work; path=/", localStorage.setItem("who", "work"), true)`, name))
	wait.Until(t, "the work page's cookie to reach its session", func() bool { return sessionCookie(t, work.Cookies, name) == "work" })
	if err := home.Cookies.Set(mygo.Cookie{Name: name, Value: "home", Domain: "127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	if got := sessionCookie(t, work.Cookies, name); got != "work" {
		t.Fatalf("the work session's cookie is %q after the home session set its own", got)
	}
	if got := sessionCookie(t, mygo.Cookies, name); got != "" {
		t.Fatalf("the default session has the cookie of another: %q", got)
	}
	waitFor(t, w2, fmt.Sprintf(`document.cookie.includes("%s=home") && !document.cookie.includes("%s=work")`, name, name))
	if got := mustEval(t, w2, `localStorage.getItem("who")`); got != nil {
		t.Fatalf("the home page reads the work page's local storage: %v", got)
	}
	if got := mustEval(t, d, fmt.Sprintf(`document.cookie.includes("%s=") || localStorage.getItem("who")`, name)); got != false && got != nil {
		t.Fatalf("the default session's page sees another session's data: %v", got)
	}
	header, err := mygo.EvalAs[string](w1.Page(), "fetch('/echo').then(r => r.text())")
	if err != nil || !strings.Contains(header, name+"=work") || strings.Contains(header, name+"=home") {
		t.Fatalf("the work page's request sent %q (%v)", header, err)
	}

	// A window the work page opens is in the work session.
	var mu sync.Mutex
	var opened *mygo.Window
	w1.Page().SetWindowOpenHandler(func(req mygo.WindowOpenRequest) *mygo.WindowOptions {
		return &mygo.WindowOptions{Title: "Opened", Width: 300, Height: 200}
	})
	off := mygo.App.OnWindowCreated(func(w *mygo.Window) {
		mu.Lock()
		opened = w
		mu.Unlock()
	})
	mustEval(t, w1, fmt.Sprintf(`(window.open(%q), true)`, url+"/"))
	wait.Until(t, "the work page's window to open", func() bool { mu.Lock(); defer mu.Unlock(); return opened != nil })
	off()
	t.Cleanup(opened.Destroy)
	waitFor(t, opened, fmt.Sprintf(`document.title === 'sessions' && document.cookie.includes("%s=work") && localStorage.getItem("who") === "work"`, name))

	// Custom schemes serve every session's pages, whose storage stays
	// apart there too.
	for _, w := range []*mygo.Window{w1, w2} {
		if err := w.Page().LoadURL("app://localhost/"); err != nil {
			t.Fatal(err)
		}
		waitFor(t, w, "window.run")
	}
	mustEval(t, w1, `(localStorage.setItem("scheme", "work"), true)`)
	if got := mustEval(t, w2, `localStorage.getItem("scheme")`); got != nil {
		t.Fatalf("the home page reads the work page's storage of app://: %v", got)
	}
	got, err := mygo.EvalAs[[]any](w2.Page(), "run()")
	if err != nil || len(got) == 0 || got[0] != "Hello, e2e!" {
		t.Fatalf("a bound method from app:// in another session: %v %v", got, err)
	}

	// The work session keeps its data once its windows closed.
	w1.Destroy()
	opened.Destroy()
	w3 := sessionWindow(t, work, url+"/", "Work again")
	if got := mustEval(t, w3, `localStorage.getItem("who")`); got != "work" {
		t.Fatalf("the work session lost its storage with its windows: %v", got)
	}

	// ClearData clears the kinds asked for.
	if err := work.ClearData(mygo.SiteDataCookies); err != nil {
		t.Fatal(err)
	}
	if got := sessionCookie(t, work.Cookies, name); got != "" {
		t.Fatalf("ClearData(SiteDataCookies) kept the cookie %q", got)
	}
	if got := sessionCookie(t, home.Cookies, name); got != "home" {
		t.Fatalf("clearing the work session cleared the home session's cookie: %q", got)
	}
	w3.Page().Reload()
	waitFor(t, w3, `document.title === 'sessions' && localStorage.getItem("who") === "work"`)
	if err := work.ClearData(mygo.SiteDataLocalStorage); err != nil {
		t.Fatal(err)
	}
	w3.Page().Reload()
	waitFor(t, w3, `document.title === 'sessions' && localStorage.getItem("who") === null`)
}

// TestEphemeralSession shows that an ephemeral session's pages share its
// data while one is open, apart from the default session, and that the
// data goes with the last of them.
func TestEphemeralSession(t *testing.T) {
	url := sessionServer(t)
	name := fmt.Sprintf("e%d", time.Now().UnixNano())
	private := mygo.NewEphemeralSession()
	if !private.IsEphemeral() || private.Name() != "" || mygo.NewEphemeralSession() == private {
		t.Fatal("NewEphemeralSession")
	}
	// A cookie set before the first page goes to it.
	if err := private.Cookies.Set(mygo.Cookie{Name: name + "early", Value: "1", Domain: "127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	w1 := sessionWindow(t, private, url+"/", "Private")
	waitFor(t, w1, fmt.Sprintf(`document.cookie.includes("%searly=1")`, name))
	mustEval(t, w1, fmt.Sprintf(`(document.cookie = "%s=private; path=/", localStorage.setItem("who", "private"), true)`, name))
	wait.Until(t, "the private page's cookie to reach its session", func() bool { return sessionCookie(t, private.Cookies, name) == "private" })
	if got := sessionCookie(t, mygo.Cookies, name); got != "" {
		t.Fatalf("the default session has the private cookie: %q", got)
	}
	w2 := sessionWindow(t, private, url+"/", "Private too")
	waitFor(t, w2, fmt.Sprintf(`document.cookie.includes("%s=private") && localStorage.getItem("who") === "private"`, name))
	if err := private.ClearData(mygo.SiteDataAll); err != nil {
		t.Fatal(err)
	}
	if got := sessionCookie(t, private.Cookies, name); got != "" {
		t.Fatalf("ClearData(SiteDataAll) kept the cookie %q", got)
	}
	w2.Page().Reload()
	waitFor(t, w2, `document.title === 'sessions' && localStorage.getItem("who") === null`)
	mustEval(t, w2, fmt.Sprintf(`(document.cookie = "%s=again; path=/", localStorage.setItem("who", "again"), true)`, name))
	wait.Until(t, "the private page's cookie to reach its session", func() bool { return sessionCookie(t, private.Cookies, name) == "again" })

	w1.Destroy()
	w2.Destroy()
	if got := sessionCookie(t, private.Cookies, name); got != "" {
		t.Fatalf("the private session kept its cookie after its last window closed: %q", got)
	}
	w3 := sessionWindow(t, private, url+"/", "Private again")
	if got := mustEval(t, w3, fmt.Sprintf(`document.cookie.includes("%s=") || localStorage.getItem("who")`, name)); got != false && got != nil {
		t.Fatalf("the private session kept its data after its last window closed: %v", got)
	}
}

// TestWebViewAvailableOffMain asks WebViewAvailable on another goroutine
// before any window shows a page, then opens one of an ephemeral session.
// On Linux the call loaded WebKitGTK there and made its first call into
// it, so that WebKitGTK took that thread for its UI thread and aborted as
// the main thread made the session (#1383). The calls run in a helper
// process, which has loaded no web view before.
func TestWebViewAvailableOffMain(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), wait.Guard)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "MYGO_E2E_WEBVIEW_OFF_MAIN=1")
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "webview off main: the page loaded\n") {
		t.Errorf("exit: %v, output %q", err, out)
	}
}

// webViewOffMain is the helper process of TestWebViewAvailableOffMain.
func webViewOffMain() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!doctype html><title>opened</title>")
	}))
	defer server.Close()
	mygo.App.WhenReady(func() {
		go func() {
			defer mygo.App.Quit()
			if !mygo.WebViewAvailable() {
				fmt.Println("webview off main: no webview")
				return
			}
			private := mygo.NewEphemeralSession()
			w := mygo.NewWindow(mygo.WindowOptions{Title: "WebView off main", Width: 200, Height: 100, URL: server.URL, Page: mygo.PageOptions{Session: private}})
			defer w.Destroy()
			// Until the page loads: the test's guard ends a hang.
			for {
				if loaded, err := mygo.EvalAs[bool](w.Page(), "document.title === 'opened'"); err == nil && loaded {
					fmt.Println("webview off main: the page loaded")
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
