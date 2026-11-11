package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandshakeRequiresRequestedSubprotocol(t *testing.T) {
	for _, tc := range []struct {
		name, header string
		protocols    []string
		wantOpen     bool
	}{
		{"missing", "", []string{"chat"}, false},
		{"empty", "Sec-WebSocket-Protocol: \r\n", []string{"chat"}, false},
		{"selected", "Sec-WebSocket-Protocol: chat\r\n", []string{"chat", "v2"}, true},
		{"not requested", "", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nc, rw, err := http.NewResponseController(w).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				defer nc.Close()
				rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + acceptKey(r.Header.Get("Sec-WebSocket-Key")) + "\r\n" + tc.header + "\r\n")
				rw.Flush()
			}))
			defer srv.Close()
			s := New(Options{}).Service.(*service)
			events, _ := open(t, s, connectOptions{ID: "test", URL: srv.URL, Protocols: tc.protocols})
			ev := next(t, events)
			if tc.wantOpen {
				if ev.Type != "open" {
					t.Fatalf("handshake: %+v", ev)
				}
			} else if ev.Type != "close" || ev.Code != 1006 || ev.Clean || ev.Error == "" {
				t.Fatalf("unacknowledged subprotocol should fail the handshake: %+v", ev)
			}
		})
	}
}
