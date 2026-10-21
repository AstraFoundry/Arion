// Package oauthloopback is the native-app half of an OAuth authorization
// code flow (RFC 8252): PKCE S256, a random state, and a one-shot HTTP
// listener on the loopback interface that receives the browser's redirect.
// The hosted-sync sign-in and the ChatGPT sign-in for cloud polishing share
// it; each owns its endpoints, client and error codes.
package oauthloopback

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"time"
)

// Callback is the redirect's query.
type Callback struct {
	State string
	Code  string
	Error string
}

// Server serves one callback.
type Server struct {
	srv *http.Server
	ch  chan Callback
}

// ErrAddress reports a listener that is not on the IPv4 loopback.
var ErrAddress = errors.New("listener is not on 127.0.0.1")

// Start listens on addr (127.0.0.1 with a port, or port 0 for a random one)
// and serves GET path. page renders the response body when it is not nil;
// otherwise the response is empty. It returns the redirect URI on
// 127.0.0.1. Cancelling ctx closes the server.
func Start(ctx context.Context, addr, path string, page func() []byte) (*Server, string, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}
	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil || host != "127.0.0.1" || port == "" {
		_ = ln.Close()
		return nil, "", ErrAddress
	}
	s := &Server{ch: make(chan Callback, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cb := Callback{State: q.Get("state"), Code: q.Get("code"), Error: q.Get("error")}
		select {
		case s.ch <- cb:
		default:
		}
		w.Header().Set("Cache-Control", "no-store")
		if page == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		// The URL carries the authorization code: no referrer, no scripts.
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'none'; base-uri 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(page())
	})
	s.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	go func() { _ = s.srv.Serve(ln) }()
	go func() {
		<-ctx.Done()
		s.Close()
	}()
	if err := waitUntilAccepting(ctx, ln.Addr().String()); err != nil {
		s.Close()
		return nil, "", err
	}
	return s, "http://127.0.0.1:" + port + path, nil
}

func waitUntilAccepting(ctx context.Context, addr string) error {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return err
		case <-tick.C:
		}
	}
}

// Wait returns the first callback, or ctx's error.
func (s *Server) Wait(ctx context.Context) (Callback, error) {
	select {
	case <-ctx.Done():
		return Callback{}, ctx.Err()
	case cb := <-s.ch:
		return cb, nil
	}
}

// Close stops the server; it is safe on nil and more than once.
func (s *Server) Close() {
	if s == nil || s.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
}

// NewPKCE returns a verifier and its S256 challenge.
func NewPKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// NewState returns a random state value.
func NewState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// MatchState compares in constant time; empty values never match.
func MatchState(expected, got string) bool {
	return expected != "" && got != "" && subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}
