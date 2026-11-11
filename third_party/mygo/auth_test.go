//go:build !js

package mygo

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestAuthHandler(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	ask := func(req platform.AuthRequest) platform.AuthResponse {
		return onMainValue(func() platform.AuthResponse { return fw.H.AuthRequested(req) })
	}
	challenge := platform.AuthRequest{Host: "router.lan", Port: 80, Realm: "admin", Scheme: "basic"}
	// Without a handler the web view decides, as before.
	if got := ask(challenge); got != (platform.AuthResponse{Default: true}) {
		t.Errorf("default answer = %+v", got)
	}
	var got AuthRequest
	w.Page().SetAuthHandler(func(req AuthRequest) AuthResponse {
		got = req
		if req.Host == "router.lan" && !req.IsProxy {
			return AuthResponse{Username: "admin", Password: "hunter2"}
		}
		return AuthResponse{}
	})
	if r := ask(challenge); r != (platform.AuthResponse{Username: "admin", Password: "hunter2"}) {
		t.Errorf("answer = %+v", r)
	}
	if got != (AuthRequest{Host: "router.lan", Port: 80, Realm: "admin", Scheme: "basic"}) {
		t.Errorf("request = %+v", got)
	}
	// An empty response cancels: no credentials, and not the default.
	proxy := platform.AuthRequest{Host: "proxy.lan", Port: 3128, Scheme: "digest", IsProxy: true}
	if r := ask(proxy); r != (platform.AuthResponse{}) {
		t.Errorf("canceled answer = %+v", r)
	}
	if !got.IsProxy || got.Scheme != "digest" || got.Port != 3128 {
		t.Errorf("proxy request = %+v", got)
	}
	w.Page().SetAuthHandler(nil)
	if r := ask(challenge); !r.Default {
		t.Error("SetAuthHandler(nil) did not restore the default")
	}
}

func TestCertificateErrorHandler(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	handles := func() bool { return onMainValue(fw.H.HandlesCertificateErrors) }
	ask := func(e platform.CertificateError) bool {
		return onMainValue(func() bool { return fw.H.CertificateError(e) })
	}
	der := []byte{0x30, 0x82, 0x01, 0x0a}
	e := platform.CertificateError{URL: "https://dev.example:8443/app", Host: "dev.example", Error: "the certificate's issuer is not known", Certificate: der}
	// By default certificate errors fail, and backends need not check.
	if handles() || ask(e) {
		t.Error("a certificate error is allowed without a handler")
	}
	var got CertificateError
	w.Page().SetCertificateErrorHandler(func(e CertificateError) bool {
		got = e
		return e.Host == "dev.example" && bytes.Equal(e.Certificate, der)
	})
	if !handles() || !ask(e) {
		t.Error("the handler's yes was not applied")
	}
	if got.URL != e.URL || got.Host != e.Host || got.Error != e.Error || !bytes.Equal(got.Certificate, der) {
		t.Errorf("error = %+v", got)
	}
	other := e
	other.Certificate = []byte{0x30, 0x00}
	if ask(other) {
		t.Error("the handler's no was not applied")
	}
	w.Page().SetCertificateErrorHandler(nil)
	if handles() || ask(e) {
		t.Error("SetCertificateErrorHandler(nil) did not restore the default")
	}
}

// The page of a web view in native UI has handlers of its own.
func TestWebViewAuthHandlers(t *testing.T) {
	show := true
	_, v, _, fv := webViewWindow(t, &show)
	v.Page().SetAuthHandler(func(AuthRequest) AuthResponse { return AuthResponse{Username: "u", Password: "p"} })
	v.Page().SetCertificateErrorHandler(func(CertificateError) bool { return true })
	if r := onMainValue(func() platform.AuthResponse { return fv.H.AuthRequested(platform.AuthRequest{Host: "h"}) }); r.Username != "u" {
		t.Errorf("web view answer = %+v", r)
	}
	if !onMainValue(func() bool { return fv.H.CertificateError(platform.CertificateError{Host: "h"}) }) {
		t.Error("the web view's certificate error handler was not asked")
	}
}
