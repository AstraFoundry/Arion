//go:build !js

package mygo

import (
	"slices"
	"testing"
	"time"
)

// getCookie returns the cookie of name, domain and path.
func getCookie(t *testing.T, name, domain, path string) (Cookie, bool) {
	t.Helper()
	list, err := Cookies.List(CookieFilter{Name: name, Domain: domain, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if c.Domain == domain {
			return c, true
		}
	}
	return Cookie{}, false
}

func TestCookiesRoundTrip(t *testing.T) {
	if err := Cookies.Clear(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Cookies.Clear() })
	empty, err := Cookies.List(CookieFilter{})
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty List = %v, %v", empty, err)
	}
	expiry := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	for _, path := range []string{"", "/account"} {
		if err := Cookies.Set(Cookie{Name: "session", Value: "secret", Domain: "EXAMPLE.COM", Path: path, HTTPOnly: true, Expires: expiry}); err != nil {
			t.Fatal(err)
		}
	}
	got, found := getCookie(t, "session", "example.com", "/")
	if !found || got.Domain != "example.com" || got.Path != "/" || got.SameSite != CookieSameSiteLax || !got.Expires.Equal(expiry) {
		t.Fatalf("Get = %+v, %v", got, found)
	}
	got.Value = "changed"
	if err := Cookies.Set(got); err != nil {
		t.Fatal(err)
	}
	if err := Cookies.Delete(Cookie{Name: "session", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, found := getCookie(t, "session", "example.com", "/"); found {
		t.Fatal("deleted cookie found")
	}
	if _, found := getCookie(t, "session", "example.com", "/account"); !found {
		t.Fatal("sibling cookie lost")
	}
	if err := Cookies.Delete(Cookie{Name: "absent", Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	// Delete takes a cookie List returned.
	list, err := Cookies.List(CookieFilter{})
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %+v, %v", list, err)
	}
	if err := Cookies.Delete(list[0]); err != nil {
		t.Fatal(err)
	}
	if list, err := Cookies.List(CookieFilter{}); err != nil || len(list) != 0 {
		t.Fatalf("List after Delete = %+v, %v", list, err)
	}
}

func TestCookiesFilter(t *testing.T) {
	if err := Cookies.Clear(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Cookies.Clear() })
	for _, c := range []Cookie{
		{Name: "a", Domain: "example.com"},
		{Name: "b", Domain: ".example.com"},
		{Name: "a", Domain: "www.example.com", Path: "/docs"},
		{Name: "s", Domain: "www.example.com", Secure: true},
		{Name: "a", Domain: "example.org"},
		{Name: "x", Domain: "notexample.com"},
	} {
		if err := Cookies.Set(c); err != nil {
			t.Fatal(err)
		}
	}
	keys := func(f CookieFilter) []string {
		t.Helper()
		list, err := Cookies.List(f)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, c := range list {
			out = append(out, c.Name+"@"+c.Domain+c.Path)
		}
		return out
	}
	for _, tt := range []struct {
		f    CookieFilter
		want []string
	}{
		{CookieFilter{}, []string{"b@.example.com/", "a@example.com/", "a@example.org/", "x@notexample.com/", "s@www.example.com/", "a@www.example.com/docs"}},
		{CookieFilter{Name: "a"}, []string{"a@example.com/", "a@example.org/", "a@www.example.com/docs"}},
		{CookieFilter{Domain: ".Example.com"}, []string{"b@.example.com/", "a@example.com/", "s@www.example.com/", "a@www.example.com/docs"}},
		{CookieFilter{Domain: "www.example.com", Path: "/docs"}, []string{"a@www.example.com/docs"}},
		{CookieFilter{URL: "http://example.com/"}, []string{"b@.example.com/", "a@example.com/"}},
		{CookieFilter{URL: "http://WWW.example.com/docs/intro"}, []string{"b@.example.com/", "a@www.example.com/docs"}},
		{CookieFilter{URL: "https://www.example.com/docsx"}, []string{"b@.example.com/", "s@www.example.com/"}},
		{CookieFilter{URL: "https://www.example.com/", Name: "s"}, []string{"s@www.example.com/"}},
	} {
		if got := keys(tt.f); !slices.Equal(got, tt.want) {
			t.Errorf("List(%+v) = %q, want %q", tt.f, got, tt.want)
		}
	}
	for _, u := range []string{"ftp://example.com/", "example.com", "http://", "::"} {
		if _, err := Cookies.List(CookieFilter{URL: u}); err == nil {
			t.Errorf("List with URL %q succeeded", u)
		}
	}
}

// TestCookiesValue sets the values browsers take.
func TestCookiesValue(t *testing.T) {
	t.Cleanup(func() { _ = Cookies.Clear() })
	for _, v := range []string{"", `a b,"c"\d`, "café", "x\ty", "=="} {
		c := Cookie{Name: "v", Value: v, Domain: "example.com"}
		if err := Cookies.Set(c); err != nil {
			t.Errorf("Set value %q: %v", v, err)
			continue
		}
		if got, _ := getCookie(t, "v", "example.com", "/"); got.Value != v {
			t.Errorf("value = %q, want %q", got.Value, v)
		}
	}
}

func TestCookiesValidation(t *testing.T) {
	base := Cookie{Name: "key", Value: "secret", Domain: "example.com"}
	tests := []struct {
		name   string
		change func(*Cookie)
	}{
		{"empty name", func(c *Cookie) { c.Name = "" }}, {"name separator", func(c *Cookie) { c.Name = "a=b" }},
		{"value injection", func(c *Cookie) { c.Value = "secret\r\nX: y" }}, {"value separator", func(c *Cookie) { c.Value = "a;b" }},
		{"value space", func(c *Cookie) { c.Value = " a" }}, {"name space", func(c *Cookie) { c.Name = "a\t" }},
		{"domain port", func(c *Cookie) { c.Domain = "example.com:80" }}, {"domain trailing dot", func(c *Cookie) { c.Domain = "example.com." }},
		{"domain double dot", func(c *Cookie) { c.Domain = "..example.com" }}, {"domain IP dot", func(c *Cookie) { c.Domain = ".127.0.0.1" }},
		{"domain Kelvin sign", func(c *Cookie) { c.Domain = "\u212a.example" }},
		{"path", func(c *Cookie) { c.Path = "relative" }}, {"path injection", func(c *Cookie) { c.Path = "/;Secure" }},
		{"enum", func(c *Cookie) { c.SameSite = "other" }}, {"insecure none", func(c *Cookie) { c.SameSite = CookieSameSiteNone }},
		{"secure prefix", func(c *Cookie) { c.Name = "__Secure-a" }}, {"host prefix", func(c *Cookie) { c.Name = "__Host-a"; c.Secure = true; c.Domain = ".example.com" }},
		{"expiration", func(c *Cookie) { c.Expires = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := base
			tt.change(&c)
			if err := Cookies.Set(c); err == nil {
				t.Fatal("accepted invalid cookie")
			}
		})
	}
}

func TestCookiesMainThread(t *testing.T) {
	RunOnMain(func() {
		c := Cookie{Name: "main", Domain: "example.com"}
		if err := Cookies.Set(c); err != nil {
			t.Error(err)
		}
		if list, err := Cookies.List(CookieFilter{Name: c.Name}); err != nil || len(list) != 1 {
			t.Errorf("List: %v %v", list, err)
		}
		if err := Cookies.Delete(c); err != nil {
			t.Error(err)
		}
		if err := Cookies.Clear(); err != nil {
			t.Error(err)
		}
	})
}

func TestCookiesPastExpiry(t *testing.T) {
	c := Cookie{Name: "past", Domain: "example.com"}
	if err := Cookies.Set(c); err != nil {
		t.Fatal(err)
	}
	c.Expires = time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := Cookies.Set(c); err != nil {
		t.Fatal(err)
	}
	if _, ok := getCookie(t, c.Name, c.Domain, "/"); ok {
		t.Fatal("expired cookie found")
	}
}
