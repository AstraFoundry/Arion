package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func noBackoff(t *testing.T) {
	saved := Backoff
	Backoff = make([]time.Duration, len(saved))
	t.Cleanup(func() { Backoff = saved })
}

// TestGetRetriesTransientFailures downloads through a host that answers
// 502 and 504, and breaks a connection off, before it answers with the
// file, as GitHub's release downloads do now and then; and gives up at
// once on failures a new request would not change.
func TestGetRetriesTransientFailures(t *testing.T) {
	noBackoff(t)
	content := []byte(strings.Repeat("the library ", 1000))
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		switch r.URL.Path {
		case "/flaky":
			switch n {
			case 1:
				http.Error(w, "bad gateway", http.StatusBadGateway)
			case 2:
				http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
			case 3:
				// Announce the whole file, send part of it, and hang up.
				w.Header().Set("Content-Length", "12000")
				w.Write(content[:100])
				conn, _, err := http.NewResponseController(w).Hijack()
				if err == nil {
					conn.Close()
				}
			default:
				w.Header().Set("Content-Length", strconv.Itoa(len(content)))
				w.Write(content)
			}
		case "/missing":
			http.NotFound(w, r)
		case "/down":
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		case "/other":
			w.Write([]byte("another file"))
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "lib")

	// Each attempt starts its progress over.
	var totals []int64
	f := File{URL: srv.URL + "/flaky", SHA256: sum(content), Progress: func(total int64) io.Writer {
		totals = append(totals, total)
		return io.Discard
	}}
	if err := Get(context.Background(), f, path); err != nil {
		t.Fatalf("after two 5xx and a connection broken off: %v", err)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != string(content) {
		t.Errorf("the file holds %d bytes, %v", len(b), err)
	}
	if n := requests.Load(); n != 4 {
		t.Errorf("%d requests, want 4", n)
	}
	if want := []int64{12000, int64(len(content))}; !slices.Equal(totals, want) {
		t.Errorf("progress totals %v, want %v", totals, want)
	}

	for _, c := range []struct {
		path, err string
		requests  int32
	}{
		{"/missing", "GET " + srv.URL + "/missing: 404 Not Found", 1},
		{"/other", srv.URL + "/other has the SHA-256 " + sum([]byte("another file")) + ", not " + sum(content), 1},
		// A host that stays down fails the download after the retries.
		{"/down", "GET " + srv.URL + "/down: 503 Service Unavailable", int32(len(Backoff)) + 1},
	} {
		requests.Store(0)
		err := Get(context.Background(), File{URL: srv.URL + c.path, SHA256: sum(content)}, path)
		if err == nil || err.Error() != c.err {
			t.Errorf("%s: %v, want %s", c.path, err, c.err)
		}
		if n := requests.Load(); n != c.requests {
			t.Errorf("%s: %d requests, want %d", c.path, n, c.requests)
		}
	}
}

// TestCachedDownloadsOnceForConcurrentProcesses has a second download of a
// file start into the cache while a first one is under way, as `go test
// ./...` runs plugins/terminal and plugins/terminal/internal/vt side by
// side and two apps may start at once: the second waits for the first and
// takes the file it put there, instead of downloading it again and
// renaming its copy onto the first's, which Windows refuses ("Access is
// denied") when the first has it open or loaded.
func TestCachedDownloadsOnceForConcurrentProcesses(t *testing.T) {
	content := []byte("the native library")
	var (
		requests    atomic.Int32
		firstAsks   = make(chan struct{})
		secondWaits = make(chan struct{})
		secondAsks  = make(chan struct{})
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			close(firstAsks)
			// Hold the first download until the second one is under way
			// too: waiting for it, or downloading it as well.
			select {
			case <-secondWaits:
			case <-secondAsks:
			}
		} else {
			close(secondAsks)
		}
		w.Write(content)
	}))
	defer srv.Close()

	var mu sync.Mutex
	var replaced []string
	savedRename, savedWaiting := rename, waiting
	defer func() { rename, waiting = savedRename, savedWaiting }()
	rename = func(from, to string) error {
		if _, err := os.Stat(to); err == nil {
			mu.Lock()
			replaced = append(replaced, to)
			mu.Unlock()
		}
		return os.Rename(from, to)
	}
	waiting = func(string) { close(secondWaits) }

	path := filepath.Join(t.TempDir(), "natives", sum(content), "libx.so")
	f := File{URL: srv.URL + "/libx.so", SHA256: sum(content)}
	first := make(chan error, 1)
	go func() { first <- Cached(context.Background(), f, path) }()
	<-firstAsks // the first holds the lock now
	if err := Cached(context.Background(), f, path); err != nil {
		t.Fatal(err)
	}
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if n := requests.Load(); n != 1 {
		t.Errorf("the file was downloaded %d times", n)
	}
	if len(replaced) > 0 {
		t.Errorf("a download was renamed onto the other's file %v", replaced)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != string(content) {
		t.Errorf("the cache holds %q, %v", b, err)
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 2 { // the file and .lock
		t.Errorf("the cache directory holds %v", entries)
	}
}
