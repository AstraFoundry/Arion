package nsis

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/download"
)

// TestRelease checks that every copy of NSIS that mygo build tries is
// the zip of its version, so that updating NSIS updates them all.
func TestRelease(t *testing.T) {
	v := Release.Version
	for _, url := range Release.URLs {
		if !strings.HasSuffix(url, "/"+v+"/nsis-"+v+".zip") && !strings.HasSuffix(url, "/nsis-"+v+"/nsis-"+v+".zip") {
			t.Errorf("%s is not NSIS %s", url, v)
		}
	}
}

// TestDownload downloads NSIS from a test server into the cache once,
// from the first copy that answers with the expected archive, and refuses
// an archive that is not the expected one.
func TestDownload(t *testing.T) {
	archive := zipArchive(t, "nsis-9.9/", "nsis-9.9/Bin/makensis.exe", "nsis-9.9/Include/MUI2.nsh")
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch r.URL.Path {
		case "/hangs/nsis-9.9.zip":
			<-r.Context().Done()
		case "/down/nsis-9.9.zip":
			http.Error(w, "origin timeout", 522)
		case "/wrong/nsis-9.9.zip":
			io.WriteString(w, "<html>Not the archive</html>")
		default:
			w.Write(archive)
		}
	}))
	defer srv.Close()
	saved, savedBackoff := Release, download.Backoff
	defer func() { Release, download.Backoff = saved, savedBackoff }()
	download.Backoff = make([]time.Duration, len(savedBackoff))
	sum := sha256.Sum256(archive)
	Release.SHA256 = hex.EncodeToString(sum[:])
	Release.URLs = []string{srv.URL + "/hangs/nsis-9.9.zip", srv.URL + "/down/nsis-9.9.zip", srv.URL + "/wrong/nsis-9.9.zip", srv.URL + "/nsis-9.9.zip"}
	// A host that takes the connection and never answers gives way to the
	// next copy, as the CLI's downloads give up after a while.
	file := func(url string) download.File {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.ResponseHeaderTimeout = 200 * time.Millisecond
		return download.File{URL: url, Client: &http.Client{Transport: transport}}
	}

	cache := t.TempDir()
	dir := filepath.Join(cache, "nsis-9.9")
	tool, err := Download(context.Background(), dir, file)
	if err != nil {
		t.Fatal(err)
	}
	if tool != filepath.Join(dir, "Bin", "makensis.exe") {
		t.Errorf("makensis = %s", tool)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "Include", "MUI2.nsh")); err != nil || string(b) != "nsis-9.9/Include/MUI2.nsh" {
		t.Errorf("Include/MUI2.nsh = %q, %v", b, err)
	}
	// A host that answers 522 is asked again before the next copy is.
	if again, err := Download(context.Background(), dir, file); err != nil || again != tool || requests.Load() != int32(3+len(download.Backoff)+1) {
		t.Errorf("downloading again = %s, %v after %d requests", again, err, requests.Load())
	}

	// Every copy fails, and the error says how.
	Release.SHA256 = strings.Repeat("0", 64)
	other := filepath.Join(cache, "nsis-other")
	_, err = Download(context.Background(), other, file)
	if err == nil {
		t.Fatal("downloaded an unexpected archive")
	}
	for _, s := range []string{
		srv.URL + "/hangs/nsis-9.9.zip\": net/http: timeout awaiting response headers",
		srv.URL + "/down/nsis-9.9.zip: 522",
		srv.URL + "/wrong/nsis-9.9.zip has the SHA-256",
		srv.URL + "/nsis-9.9.zip has the SHA-256",
	} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("the error does not say %q:\n%v", s, err)
		}
	}
	var names []string
	if entries, err := os.ReadDir(cache); err == nil {
		for _, e := range entries {
			names = append(names, e.Name())
		}
	}
	if want := []string{"nsis-9.9", "nsis-9.9.lock", "nsis-other.lock"}; !slices.Equal(names, want) {
		t.Errorf("the cache holds %v, want %v", names, want)
	}
}

// TestUnzipTop unpacks the top directory of archives, and refuses entries
// outside it.
func TestUnzipTop(t *testing.T) {
	for _, tc := range []struct {
		entries []string
		ok      bool
	}{
		{[]string{"top/", "top/a/b.txt", "top/c.txt"}, true},
		{[]string{"top/a.txt", "other/b.txt"}, false},
		{[]string{"top/a.txt", "b.txt"}, false},
		{[]string{"top/../evil.txt"}, false},
		{[]string{"top/a/../../evil.txt"}, false},
	} {
		path := filepath.Join(t.TempDir(), "archive.zip")
		if err := os.WriteFile(path, zipArchive(t, tc.entries...), 0o644); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(t.TempDir(), "out")
		err := unzipTop(path, dir)
		if (err == nil) != tc.ok {
			t.Errorf("%q: %v", tc.entries, err)
			continue
		}
		if tc.ok {
			for _, name := range []string{"a/b.txt", "c.txt"} {
				if b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name))); err != nil || string(b) != "top/"+name {
					t.Errorf("%s = %q, %v", name, b, err)
				}
			}
		}
	}
}

// zipArchive makes a zip archive of the named entries: directories end with
// a slash, and files hold their names.
func zipArchive(t *testing.T, names ...string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(name, "/") {
			io.WriteString(w, name)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
