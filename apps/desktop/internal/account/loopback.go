package account

import (
	"context"
	"net"
	"net/http"
	"time"
)

type callback struct {
	state string
	code  string
	error string
}

type loopback struct {
	srv *http.Server
	ch  chan callback
}

// startLoopback serves one callback. page renders the response body when it
// is not nil; otherwise the response is empty.
func startLoopback(ctx context.Context, page func() []byte) (*loopback, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", wrap(CodeBrowser, "listen", err)
	}
	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil || host != "127.0.0.1" || port == "" {
		_ = ln.Close()
		return nil, "", &Error{Code: CodeBrowser, Detail: "loopback"}
	}
	lb := &loopback{ch: make(chan callback, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+callbackPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cb := callback{state: q.Get("state"), code: q.Get("code"), error: q.Get("error")}
		select {
		case lb.ch <- cb:
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
	lb.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	go func() { _ = lb.srv.Serve(ln) }()
	go func() {
		<-ctx.Done()
		lb.Close()
	}()
	if err := waitUntilAccepting(ctx, ln.Addr().String()); err != nil {
		lb.Close()
		return nil, "", err
	}
	return lb, "http://127.0.0.1:" + port + callbackPath, nil
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
			return wrap(CodeBrowser, "listen", err)
		case <-tick.C:
		}
	}
}

func (l *loopback) Wait(ctx context.Context) (callback, error) {
	select {
	case <-ctx.Done():
		return callback{}, ctx.Err()
	case cb := <-l.ch:
		return cb, nil
	}
}

func (l *loopback) Close() {
	if l == nil || l.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = l.srv.Shutdown(ctx)
}
