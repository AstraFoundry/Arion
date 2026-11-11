package schemehost

import "testing"

func schemes(names ...string) func(string) bool {
	return func(s string) bool {
		for _, n := range names {
			if n == s {
				return true
			}
		}
		return false
	}
}

func TestWebURL(t *testing.T) {
	has := schemes("myapp", "mygo", "com.example.app")
	for _, c := range []struct{ in, want string }{
		{"myapp://localhost/index.html?a=1#x", "http://myapp.localhost/index.html?a=1#x"},
		{"myapp://abc/page", "http://abc.myapp.localhost/page"},
		{"myapp://ABC/Page", "http://abc.myapp.localhost/Page"},
		{"myapp://a-1.b2/", "http://a-1.b2.myapp.localhost/"},
		{"myapp://0123456789abcdef/x", "http://0123456789abcdef.myapp.localhost/x"},
		{"com.example.app://localhost/", "http://com.example.app.localhost/"},
		{"com.example.app://win/", "http://win.com.example.app.localhost/"},
		// Not mapped.
		{"myapp://abc:8080/", "myapp://abc:8080/"},
		{"myapp://localhost:1/", "myapp://localhost:1/"},
		{"myapp://u@abc/", "myapp://u@abc/"},
		{"myapp:///path", "myapp:///path"},
		{"myapp:opaque", "myapp:opaque"},
		{"myapp://-abc/", "myapp://-abc/"},
		{"myapp://ab_c/", "myapp://ab_c/"},
		{"myapp://a..b/", "myapp://a..b/"},
		{"myapp://%C3%A9t%C3%A9/", "myapp://%C3%A9t%C3%A9/"},
		{"other://abc/", "other://abc/"},
		{"https://abc/", "https://abc/"},
	} {
		if got := WebURL(c.in, has); got != c.want {
			t.Errorf("WebURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWebURLAmbiguous(t *testing.T) {
	// b.a://x/ and a://x.b/ would share http://x.b.a.localhost/: the longer
	// scheme keeps it.
	has := schemes("a", "b.a")
	if got := WebURL("b.a://x/", has); got != "http://x.b.a.localhost/" {
		t.Errorf("WebURL(b.a://x/) = %q", got)
	}
	if got := WebURL("a://x.b/", has); got != "a://x.b/" {
		t.Errorf("WebURL(a://x.b/) = %q, want it unmapped", got)
	}
	if got := WebURL("a://b/", has); got != "a://b/" { // http://b.a.localhost/ is b.a://localhost/
		t.Errorf("WebURL(a://b/) = %q, want it unmapped", got)
	}
}

func TestAppURL(t *testing.T) {
	has := schemes("myapp", "com.example.app")
	for _, c := range []struct{ in, want string }{
		{"http://myapp.localhost/index.html?a=1#x", "myapp://localhost/index.html?a=1#x"},
		{"http://abc.myapp.localhost/page", "myapp://abc/page"},
		{"http://ABC.MyApp.localhost/page", "myapp://abc/page"},
		{"http://a.b.myapp.localhost/", "myapp://a.b/"},
		{"http://com.example.app.localhost/", "com.example.app://localhost/"},
		{"http://win.com.example.app.localhost/", "com.example.app://win/"},
		// Not ours.
		{"http://other.localhost/", "http://other.localhost/"},
		{"http://abc.other.localhost/", "http://abc.other.localhost/"},
		{"http://localhost/", "http://localhost/"},
		{"http://myapp.localhost:8080/", "http://myapp.localhost:8080/"},
		{"http://abc.myapp.localhost:8080/", "http://abc.myapp.localhost:8080/"},
		{"https://abc.myapp.localhost/", "https://abc.myapp.localhost/"},
		{"http://myapp.example.com/", "http://myapp.example.com/"},
		{"http://a_b.myapp.localhost/", "http://a_b.myapp.localhost/"},
	} {
		if got := AppURL(c.in, has); got != c.want {
			t.Errorf("AppURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	has := schemes("myapp", "com.example.app")
	for _, u := range []string{
		"myapp://localhost/",
		"myapp://abc/page?q=1#h",
		"myapp://a.b.c/x/y",
		"myapp://localhost.abc/",
		"com.example.app://win-2/",
	} {
		web := WebURL(u, has)
		if web == u {
			t.Errorf("WebURL(%q) not mapped", u)
		}
		if got := AppURL(web, has); got != u {
			t.Errorf("AppURL(WebURL(%q)) = %q", u, got)
		}
	}
}

func TestAppOrigin(t *testing.T) {
	has := schemes("myapp", "mygo")
	for _, c := range []struct{ in, want string }{
		{"http://mygo.localhost", "mygo://localhost"},
		{"http://myapp.localhost", "myapp://localhost"},
		{"http://abc.myapp.localhost", "myapp://abc"},
		{"http://a.b.myapp.localhost", "myapp://a.b"},
		{"http://abc.myapp.localhost:5173", "http://abc.myapp.localhost:5173"},
		{"http://abc.other.localhost", "http://abc.other.localhost"},
		{"https://abc.myapp.localhost", "https://abc.myapp.localhost"},
		{"http://localhost:5173", "http://localhost:5173"},
		{"null", "null"},
	} {
		if got := AppOrigin(c.in, has); got != c.want {
			t.Errorf("AppOrigin(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
