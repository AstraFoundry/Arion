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
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// getCookie returns the cookie of name, domain and path.
func getCookie(name, domain, path string) (mygo.Cookie, bool, error) {
	if path == "" {
		path = "/"
	}
	list, err := mygo.Cookies.List(mygo.CookieFilter{Name: name, Domain: domain, Path: path})
	for _, c := range list {
		if c.Domain == domain {
			return c, true, err
		}
	}
	return mygo.Cookie{}, false, err
}

func TestCookies(t *testing.T) {
	prefix := fmt.Sprintf("mygo%d", time.Now().UnixNano())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/seed" {
			http.SetCookie(w, &http.Cookie{Name: prefix + "server", Value: "private", Path: "/", HttpOnly: true})
		}
		if r.URL.Path == "/echo" {
			fmt.Fprint(w, r.Header.Get("Cookie"))
			return
		}
		fmt.Fprint(w, "<!doctype html><title>cookies</title><body>cookies</body>")
	}))
	defer server.Close()
	w := newWindow(t, mygo.WindowOptions{Title: "Cookies", Width: 300, Height: 200})
	if err := w.Page().LoadURL(server.URL + "/seed"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, w, "document.title === 'cookies'")
	t.Cleanup(func() {
		for _, suffix := range []string{"server", "plain", "hidden", "deep", "persistent"} {
			_ = mygo.Cookies.Delete(mygo.Cookie{Name: prefix + suffix, Domain: "127.0.0.1"})
			_ = mygo.Cookies.Delete(mygo.Cookie{Name: prefix + suffix, Domain: "127.0.0.1", Path: "/deep/path"})
		}
	})
	c, found, err := getCookie(prefix+"server", "127.0.0.1", "/")
	if err != nil || !found || !c.HTTPOnly {
		t.Fatalf("server cookie: %+v %v %v", c, found, err)
	}
	for _, suffix := range []string{"plain", "hidden", "deep", "persistent"} {
		c := mygo.Cookie{Name: prefix + suffix, Value: "value", Domain: "127.0.0.1", HTTPOnly: suffix == "hidden"}
		if suffix == "deep" {
			c.Path = "/deep/path"
		}
		if suffix == "persistent" {
			c.Expires = time.Now().Add(time.Hour).UTC().Truncate(time.Second)
		}
		if err := mygo.Cookies.Set(c); err != nil {
			t.Fatal(err)
		}
		got, found, err := getCookie(c.Name, c.Domain, c.Path)
		if err != nil || !found || got.Expires.IsZero() != c.Expires.IsZero() {
			t.Fatalf("roundtrip %s: %+v %v %v", suffix, got, found, err)
		}
	}
	visible, err := mygo.EvalAs[string](w.Page(), "document.cookie")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(visible, prefix+"plain=") || strings.Contains(visible, prefix+"hidden=") || strings.Contains(visible, prefix+"server=") || strings.Contains(visible, prefix+"deep=") {
		t.Fatalf("document.cookie visibility wrong: %q", visible)
	}
	header, err := mygo.EvalAs[string](w.Page(), "fetch('/echo').then(r=>r.text())")
	if err != nil || !strings.Contains(header, prefix+"hidden=") {
		t.Fatalf("HTTPOnly request missing: %q %v", header, err)
	}
	// A URL chooses the cookies a request to it sends, as the engine does.
	sent, err := mygo.Cookies.List(mygo.CookieFilter{URL: server.URL + "/echo"})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range sent {
		if strings.HasPrefix(c.Name, prefix) {
			names = append(names, strings.TrimPrefix(c.Name, prefix))
		}
	}
	if strings.Join(names, " ") != "hidden persistent plain server" {
		t.Fatalf("cookies for /echo: %q", names)
	}
	deep, err := mygo.Cookies.List(mygo.CookieFilter{URL: server.URL + "/deep/path/x", Name: prefix + "deep"})
	if err != nil || len(deep) != 1 {
		t.Fatalf("cookies for /deep/path/x: %+v %v", deep, err)
	}
	sibling := newWindow(t, mygo.WindowOptions{Title: "Cookie sibling", Width: 300, Height: 200})
	if err := sibling.Page().LoadURL(server.URL); err != nil {
		t.Fatal(err)
	}
	waitFor(t, sibling, "document.title === 'cookies'")
	visible, err = mygo.EvalAs[string](sibling.Page(), "document.cookie")
	if err != nil || !strings.Contains(visible, prefix+"plain=") {
		t.Fatalf("shared cookie: %q %v", visible, err)
	}
	w.Destroy()
	if err := mygo.Cookies.Delete(mygo.Cookie{Name: prefix + "plain", Domain: "127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	if _, found, err := getCookie(prefix+"plain", "127.0.0.1", ""); err != nil || found {
		t.Fatalf("deleted cookie still there: %v %v", found, err)
	}
	if _, found, err := getCookie(prefix+"deep", "127.0.0.1", "/deep/path"); err != nil || !found {
		t.Fatalf("deep sibling lost: %v %v", found, err)
	}
	// A domain cookie of a site no page has opened, with every attribute.
	want := mygo.Cookie{Name: prefix + "domain", Value: "v1", Domain: ".mygo-cookies.test", Path: "/",
		Expires: time.Now().Add(time.Hour).UTC().Truncate(time.Second), Secure: true, HTTPOnly: true, SameSite: mygo.CookieSameSiteNone}
	for _, site := range []mygo.CookieSameSite{mygo.CookieSameSiteNone, mygo.CookieSameSiteStrict, mygo.CookieSameSiteLax} {
		want.SameSite = site
		if err := mygo.Cookies.Set(want); err != nil {
			t.Fatalf("Set %s: %v", site, err)
		}
		got, found, err := getCookie(want.Name, want.Domain, want.Path)
		if err != nil || !found || got != want {
			t.Fatalf("domain cookie, SameSite %s: %+v %v %v, want %+v", site, got, found, err, want)
		}
	}
	// The host's cookie of the same name is another cookie.
	host := mygo.Cookie{Name: want.Name, Value: "v2", Domain: "mygo-cookies.test"}
	if err := mygo.Cookies.Set(host); err != nil {
		t.Fatal(err)
	}
	if err := mygo.Cookies.Delete(host); err != nil {
		t.Fatal(err)
	}
	if _, found, err := getCookie(host.Name, host.Domain, ""); err != nil || found {
		t.Fatalf("deleted host cookie still there: %v %v", found, err)
	}
	if got, found, err := getCookie(want.Name, want.Domain, want.Path); err != nil || !found || got.Value != want.Value {
		t.Fatalf("deleting the host's cookie changed the domain's: %+v %v %v", got, found, err)
	}
	// Values take what browsers take.
	for _, v := range []string{`a b,"c"\d`, "caf\u00e9", "x=y, Path=/z"} {
		want.Value = v
		if err := mygo.Cookies.Set(want); err != nil {
			t.Fatalf("Set value %q: %v", v, err)
		}
		if got, found, err := getCookie(want.Name, want.Domain, want.Path); err != nil || !found || got.Value != v {
			t.Fatalf("value %q: %+v %v %v", v, got, found, err)
		}
	}
	if err := mygo.Cookies.Delete(want); err != nil {
		t.Fatal(err)
	}
	if _, found, err := getCookie(want.Name, want.Domain, want.Path); err != nil || found {
		t.Fatalf("deleted domain cookie still there: %v %v", found, err)
	}
	if _, err := sibling.Page().Eval("localStorage.setItem('cookies-test','keep')"); err != nil {
		t.Fatal(err)
	}
	if err := mygo.Cookies.Clear(); err != nil {
		t.Fatal(err)
	}
	storage, err := mygo.EvalAs[string](sibling.Page(), "localStorage.getItem('cookies-test')")
	if err != nil || storage != "keep" {
		t.Fatalf("cookie Clear changed localStorage: %q %v", storage, err)
	}
	if err := mygo.App.ClearBrowsingData(); err != nil {
		t.Fatal(err)
	}
	sibling.Page().Reload()
	waitFor(t, sibling, "document.title === 'cookies' && localStorage.getItem('cookies-test') === null")
}

func TestCookieCallbackStress(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Cookie callback stress", Width: 200, Height: 100})
	w.Page().LoadHTML("<title>cookie stress</title>", "")
	waitFor(t, w, "document.title === 'cookie stress'")
	for i := 0; i < 2100; i++ {
		if _, err := mygo.Cookies.List(mygo.CookieFilter{}); err != nil {
			t.Fatalf("List %d: %v", i, err)
		}
	}
}

// TestCookiesBeforePages sets a cookie before a window shows a page, which
// its first request sends. On Windows, without another window showing a
// page, the cookie goes through a hidden web view.
func TestCookiesBeforePages(t *testing.T) {
	name := fmt.Sprintf("mygo%dearly", time.Now().UnixNano())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie(name)
		v := ""
		if c != nil {
			v = c.Value
		}
		fmt.Fprintf(w, "<!doctype html><title>early:%s</title>", v)
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "http://")
	host = host[:strings.LastIndex(host, ":")]
	if err := mygo.Cookies.Set(mygo.Cookie{Name: name, Value: "first", Domain: host}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mygo.Cookies.Delete(mygo.Cookie{Name: name, Domain: host}) })
	w := newWindow(t, mygo.WindowOptions{Title: "Early cookie", Width: 200, Height: 100, URL: server.URL})
	waitFor(t, w, "document.title === 'early:first'")
}

// TestCookieCallsWhileWindowOpens reads cookies as a window is made, before
// any window shows a page, so on Windows the calls take the hidden web view
// while the window's own web view is being made: closing that view from the
// callback that finished the calls crashed the message loop. The test
// process has windows showing pages from TestMain, so the calls run in a
// helper process with none.
func TestCookieCallsWhileWindowOpens(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "MYGO_E2E_COOKIES_WHILE_OPENING=1")
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "cookies: the page loaded\n") {
		t.Errorf("exit: %v, output %q", err, out)
	}
}

// cookiesWhileOpening is the helper process of TestCookieCallsWhileWindowOpens.
func cookiesWhileOpening() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!doctype html><title>opened</title>")
	}))
	defer server.Close()
	mygo.App.WhenReady(func() {
		go func() {
			defer mygo.App.Quit()
			w := mygo.NewWindow(mygo.WindowOptions{Title: "Cookies while opening", Width: 200, Height: 100, URL: server.URL})
			defer w.Destroy()
			for i := 0; i < 16; i++ {
				if _, err := mygo.Cookies.List(mygo.CookieFilter{}); err != nil {
					fmt.Printf("cookies: List %d: %v\n", i, err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
			for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
				if loaded, err := mygo.EvalAs[bool](w.Page(), "document.title === 'opened'"); err == nil && loaded {
					fmt.Println("cookies: the page loaded")
					return
				}
			}
			fmt.Println("cookies: the page did not load")
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
