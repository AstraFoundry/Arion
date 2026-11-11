//go:build !js

package mygo

import (
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
)

// storeOf returns the fake store of a fake window's page, nil for the
// default one.
func storeOf(fw *fake.Window) *fake.DataStore {
	return onMainValue(func() *fake.DataStore {
		s, _ := fw.Opts.DataStore.(*fake.DataStore)
		return s
	})
}

func TestSessionStores(t *testing.T) {
	work := NewSession("unit-work")
	if NewSession("unit-work") != work || work.Name() != "unit-work" || work.IsEphemeral() {
		t.Fatal("NewSession of the same name made another session")
	}
	if NewSession("unit-home") == work {
		t.Fatal("NewSession of another name returned the same session")
	}
	if d := DefaultSession(); d.Name() != "" || d.IsEphemeral() || d.Cookies != Cookies {
		t.Fatal("DefaultSession")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("NewSession(\"\") did not panic")
			}
		}()
		NewSession("")
	}()

	w1, fw1 := testWindow(t, WindowOptions{Page: PageOptions{Session: work}})
	_, fw2 := testWindow(t, WindowOptions{Page: PageOptions{Session: work}})
	plainWindow, plain := testWindow(t, WindowOptions{})
	_, def := testWindow(t, WindowOptions{Page: PageOptions{Session: DefaultSession()}})
	store := storeOf(fw1)
	if store == nil || storeOf(fw2) != store {
		t.Fatalf("the pages of a session have stores %v and %v", store, storeOf(fw2))
	}
	if storeOf(plain) != nil || storeOf(def) != nil {
		t.Fatal("a page of the default session got a store")
	}
	if store.Opts.Ephemeral || store.Opts.ID != sessionID("unit-work") {
		t.Fatalf("store options %+v", store.Opts)
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-8[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(store.Opts.ID) || sessionID("unit-home") == store.Opts.ID {
		t.Fatalf("session ID %q", store.Opts.ID)
	}
	if runtime.GOOS == "linux" {
		if !strings.HasSuffix(store.Opts.DataDir, "/Sessions/"+store.Opts.ID) || !strings.HasSuffix(store.Opts.CacheDir, "/Sessions/"+store.Opts.ID) || store.Opts.DataDir == store.Opts.CacheDir {
			t.Fatalf("directories %q and %q", store.Opts.DataDir, store.Opts.CacheDir)
		}
	}

	// Cookies of a session are its own.
	t.Cleanup(func() { _ = Cookies.Clear() })
	if err := work.Cookies.Set(Cookie{Name: "who", Value: "work", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := Cookies.Set(Cookie{Name: "who", Value: "default", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		m    CookiesModule
		want string
	}{{work.Cookies, "work"}, {Cookies, "default"}, {DefaultSession().Cookies, "default"}} {
		list, err := c.m.List(CookieFilter{Name: "who"})
		if err != nil || len(list) != 1 || list[0].Value != c.want {
			t.Fatalf("List = %v, %v, want %s", list, err, c.want)
		}
	}
	// A cookie that expired deletes the session's own.
	if err := work.Cookies.Set(Cookie{Name: "who", Domain: "example.com", Expires: tPast}); err != nil {
		t.Fatal(err)
	}
	if list, _ := work.Cookies.List(CookieFilter{}); len(list) != 0 {
		t.Fatalf("expired cookie kept: %v", list)
	}
	if list, _ := Cookies.List(CookieFilter{}); len(list) != 1 {
		t.Fatalf("the default session lost its cookie: %v", list)
	}

	// ClearData clears the session's store, of the kinds asked.
	if err := work.Cookies.Set(Cookie{Name: "who", Value: "work", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	cleared := onMainValue(func() int { return fb.Cleared })
	if err := work.ClearData(0); err != nil {
		t.Fatal(err)
	}
	if err := work.ClearData(SiteDataLocalStorage | SiteDataCaches); err != nil {
		t.Fatal(err)
	}
	if list, _ := work.Cookies.List(CookieFilter{}); len(list) != 1 {
		t.Fatalf("ClearData without cookies cleared them: %v", list)
	}
	if err := work.ClearData(SiteDataCookies); err != nil {
		t.Fatal(err)
	}
	if list, _ := work.Cookies.List(CookieFilter{}); len(list) != 0 {
		t.Fatalf("ClearData(SiteDataCookies) kept %v", list)
	}
	if got := onMainValue(func() []platform.SiteData { return store.Cleared }); len(got) != 2 || got[0] != platform.SiteDataLocalStorage|platform.SiteDataCaches || got[1] != platform.SiteDataCookies {
		t.Fatalf("the store cleared %v", got)
	}
	if list, _ := Cookies.List(CookieFilter{}); len(list) != 1 || onMainValue(func() int { return fb.Cleared }) != cleared {
		t.Fatal("clearing a session cleared the default one")
	}
	if err := DefaultSession().ClearData(SiteDataAll); err != nil {
		t.Fatal(err)
	}
	if list, _ := Cookies.List(CookieFilter{}); len(list) != 0 || onMainValue(func() int { return fb.Cleared }) != cleared+1 {
		t.Fatal("DefaultSession().ClearData did not clear the default store")
	}
	if !platform.SiteData(SiteDataAll).All() || platform.SiteData(SiteDataCookies|SiteDataLocalStorage|SiteDataIndexedDB|SiteDataCaches|SiteDataServiceWorkers).All() {
		t.Fatal("SiteDataAll is not apart from the kinds it has")
	}

	// A window a page opens is in its opener's session, whatever its
	// options say.
	w1.Page().SetWindowOpenHandler(func(WindowOpenRequest) *WindowOptions {
		return &WindowOptions{Page: PageOptions{Session: NewSession("unit-home")}}
	})
	child := onMainValue(func() platform.Window {
		return fw1.H.NewWindow(platform.NewWindowRequest{URL: "https://example.com/", UserInitiated: true})
	})
	if child == nil {
		t.Fatal("the window did not open")
	}
	fc := child.(*fake.Window)
	t.Cleanup(func() { onMain(fc.Close) })
	if storeOf(fc) != store {
		t.Fatal("the opened window is not in its opener's session")
	}
	plainWindow.Page().SetWindowOpenHandler(func(WindowOpenRequest) *WindowOptions {
		return &WindowOptions{Page: PageOptions{Session: work}}
	})
	child = onMainValue(func() platform.Window {
		return plain.H.NewWindow(platform.NewWindowRequest{URL: "https://example.com/", UserInitiated: true})
	})
	fd := child.(*fake.Window)
	t.Cleanup(func() { onMain(fd.Close) })
	if storeOf(fd) != nil {
		t.Fatal("a window the default session's page opened is in another session")
	}

	// A persistent session keeps its store.
	for _, fw := range []*fake.Window{fw1, fw2, fc} {
		onMain(fw.Close)
	}
	if onMainValue(func() bool { return store.Closed }) {
		t.Fatal("the store of a persistent session closed with its pages")
	}
	_, fw3 := testWindow(t, WindowOptions{Page: PageOptions{Session: work}})
	if storeOf(fw3) != store {
		t.Fatal("a persistent session's page got another store")
	}
}

// tPast is a time long gone, for cookies that expired.
var tPast = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

func TestEphemeralSessionStore(t *testing.T) {
	private := NewEphemeralSession()
	if !private.IsEphemeral() || private.Name() != "" || NewEphemeralSession() == private {
		t.Fatal("NewEphemeralSession")
	}
	// A cookie set before the first page makes the store the page gets.
	if err := private.Cookies.Set(Cookie{Name: "early", Value: "1", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	_, fw1 := testWindow(t, WindowOptions{Page: PageOptions{Session: private}})
	store := storeOf(fw1)
	if store == nil || !store.Opts.Ephemeral || store.Opts.ID != "" || store.Opts.DataDir != "" {
		t.Fatalf("store %+v", store)
	}
	if list, _ := private.Cookies.List(CookieFilter{}); len(list) != 1 {
		t.Fatalf("the cookie set before the first page is gone: %v", list)
	}
	_, fw2 := testWindow(t, WindowOptions{Page: PageOptions{Session: private}})
	if storeOf(fw2) != store {
		t.Fatal("the pages of an ephemeral session have different stores")
	}
	if list, _ := Cookies.List(CookieFilter{Name: "early"}); len(list) != 0 {
		t.Fatalf("the default session has the ephemeral one's cookie: %v", list)
	}

	// A web view in native UI uses the session too.
	show := true
	_, v, _, _ := webViewWindowOpts(t, &show, WebViewOptions{Page: PageOptions{Session: private}})
	if onMainValue(func() platform.DataStore { return v.w.native.(*webViewNative).web.(*fake.WebView).Opts.DataStore }) != store {
		t.Fatal("the web view is not in the session")
	}

	onMain(fw1.Close)
	onMain(fw2.Close)
	if onMainValue(func() bool { return store.Closed }) {
		t.Fatal("the store closed while a web view used it")
	}
	v.Destroy()
	if !onMainValue(func() bool { return store.Closed }) {
		t.Fatal("the store of an ephemeral session outlived its last page")
	}
	if list, err := private.Cookies.List(CookieFilter{}); err != nil || len(list) != 0 {
		t.Fatalf("the ephemeral session kept %v (%v)", list, err)
	}
	_, fw3 := testWindow(t, WindowOptions{Page: PageOptions{Session: private}})
	if s := storeOf(fw3); s == nil || s == store {
		t.Fatal("the ephemeral session's next page did not get a new store")
	}
}
