//go:build !js

package e2e

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// offer sends v on ch unless ch is full: handlers run on the main thread,
// which must not wait for the test, and macOS asks once per connection.
func offer[T any](ch chan T, v T) {
	select {
	case ch <- v:
	default:
	}
}

// basicAuthServer serves a page to alice with her password, and its
// refusal to the others.
func basicAuthServer(realm string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); ok && user == "alice" && pass == "s3cret" {
			fmt.Fprintf(w, "<!doctype html><title>welcome %s</title>", user)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "<!doctype html><title>denied</title>")
	}))
}

func TestAuthHandler(t *testing.T) {
	server := basicAuthServer("mygo e2e")
	defer server.Close()
	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())

	w := newWindow(t, mygo.WindowOptions{Title: "Auth", Width: 300, Height: 200})
	asked := make(chan mygo.AuthRequest, 8)
	w.Page().SetAuthHandler(func(req mygo.AuthRequest) mygo.AuthResponse {
		offer(asked, req)
		return mygo.AuthResponse{Username: "alice", Password: "s3cret"}
	})
	if err := w.Page().LoadURL(server.URL + "/"); err != nil {
		t.Fatal(err)
	}
	req := wait.For(t, "the auth handler", asked)
	want := mygo.AuthRequest{Host: "127.0.0.1", Port: port, Realm: "mygo e2e", Scheme: "basic"}
	if req != want {
		t.Errorf("request = %+v, want %+v", req, want)
	}
	waitFor(t, w, `document.title === "welcome alice"`)

	// An empty answer cancels: the page gets the server's refusal.
	other := basicAuthServer("elsewhere")
	defer other.Close()
	w.Page().SetAuthHandler(func(req mygo.AuthRequest) mygo.AuthResponse {
		offer(asked, req)
		return mygo.AuthResponse{}
	})
	if err := w.Page().LoadURL(other.URL + "/"); err != nil {
		t.Fatal(err)
	}
	if req := wait.For(t, "the auth handler", asked); req.Realm != "elsewhere" {
		t.Errorf("request = %+v", req)
	}
	waitFor(t, w, `document.title === "denied"`)
}

// selfSigned returns a certificate for 127.0.0.1 that no one issued.
func selfSigned(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "mygo e2e"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func TestCertificateErrorHandler(t *testing.T) {
	cert := selfSigned(t)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!doctype html><title>trusted</title>")
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	server.StartTLS()
	defer server.Close()

	// A window for each answer: after a failed load, WebKitGTK loads
	// its error page, which would replace the next load.
	open := func(handler func(mygo.CertificateError) bool) (*mygo.Window, chan *mygo.LoadError) {
		t.Helper()
		w := newWindow(t, mygo.WindowOptions{Title: "Certificates", Width: 300, Height: 200})
		failed := make(chan *mygo.LoadError, 8)
		w.Page().OnDidFailLoad(func(e *mygo.LoadError) { offer(failed, e) })
		if handler != nil {
			w.Page().SetCertificateErrorHandler(handler)
		}
		if err := w.Page().LoadURL(server.URL + "/"); err != nil {
			t.Fatal(err)
		}
		return w, failed
	}

	// By default the load fails.
	_, failed := open(nil)
	if e := wait.For(t, "the load to fail without a handler", failed); !strings.HasPrefix(e.URL, server.URL) {
		t.Errorf("failed load = %+v", e)
	}

	// The handler is asked, and a no fails the load too.
	asked := make(chan mygo.CertificateError, 8)
	w, failed := open(func(e mygo.CertificateError) bool {
		offer(asked, e)
		return false
	})
	e := wait.For(t, "the certificate error handler", asked)
	if e.Host != "127.0.0.1" || !strings.HasPrefix(e.URL, server.URL) || e.Error == "" {
		t.Errorf("certificate error = %+v", e)
	}
	if !bytes.Equal(e.Certificate, cert.Certificate[0]) {
		t.Error("the handler got another certificate than the server's")
	}
	wait.For(t, "the load to fail after the handler said no", failed)
	if title := mustEval(t, w, `document.title`); title == "trusted" {
		t.Fatal("the page loaded although the handler said no")
	}

	// A yes loads it.
	w, _ = open(func(e mygo.CertificateError) bool {
		return e.Host == "127.0.0.1" && bytes.Equal(e.Certificate, cert.Certificate[0])
	})
	waitFor(t, w, `document.title === "trusted"`)
}
