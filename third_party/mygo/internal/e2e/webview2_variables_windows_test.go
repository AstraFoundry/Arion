//go:build windows && (amd64 || arm64)

package e2e

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// Without a loader DLL next to the executable, as here, MyGo loads the
// WebView2 Runtime itself, and a page's browser still takes
// WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS and keeps its data in
// WEBVIEW2_USER_DATA_FOLDER, as Microsoft's loader gives them: another
// process reaches the page over the DevTools protocol, as tests and agents
// do (#419).
func TestWebView2LoaderVariables(t *testing.T) {
	if os.Getenv("MYGO_E2E_WEBVIEW2_VARIABLES") != "1" {
		// WebView2 reads these when the helper's first page creates its
		// environment, during TestMain's warm-up.
		// Not t.TempDir: the page's browser processes outlive the helper
		// for a moment and keep files of the folder open, which fails
		// TempDir's removal. Remove it once they let go.
		folder, err := os.MkdirTemp("", "mygo-webview2-variables-")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			wait.Until(t, "WebView2's browser processes to let go of "+folder, func() bool { return os.RemoveAll(folder) == nil })
		})
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestWebView2LoaderVariables$", "-test.v")
		cmd.Env = append(os.Environ(),
			"MYGO_E2E_WEBVIEW2_VARIABLES=1",
			"WEBVIEW2_USER_DATA_FOLDER="+folder,
			"WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=0",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v:\n%s", err, out)
		}
		return
	}

	const title = "MyGo WebView2 loader variables E2E"
	w := newWindow(t, mygo.WindowOptions{Title: title, Width: 300, Height: 200})
	w.Page().LoadHTML("<title>"+title+"</title>", "")
	waitFor(t, w, fmt.Sprintf("document.title === %q", title))

	folder := os.Getenv("WEBVIEW2_USER_DATA_FOLDER")
	var portFile string
	var port int
	var fileErr error
	defer func() {
		if t.Failed() {
			if portFile == "" {
				t.Logf("no DevToolsActivePort file under %q: %v", folder, fileErr)
			} else {
				t.Logf("DevToolsActivePort file %q: port %d, last error: %v", portFile, port, fileErr)
			}
		}
	}()
	wait.Until(t, fmt.Sprintf("a DevToolsActivePort file with a port under %q", folder), func() bool {
		// WebView2 keeps its browser's data under this folder; the
		// layout within it belongs to the runtime.
		if err := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || entry.Name() != "DevToolsActivePort" {
				return nil
			}
			portFile = path
			data, err := os.ReadFile(path)
			if err != nil {
				fileErr = err
				return nil
			}
			line, _, _ := strings.Cut(string(data), "\n")
			n, err := strconv.Atoi(strings.TrimSpace(line))
			if err != nil || n < 1 || n > 65535 {
				fileErr = fmt.Errorf("invalid port %q: %v", line, err)
				return nil
			}
			port, fileErr = n, nil
			return filepath.SkipAll
		}); err != nil {
			fileErr = err
		}
		return port != 0
	})

	url := fmt.Sprintf("http://127.0.0.1:%d/json/list", port)
	req, err := http.NewRequestWithContext(wait.Context(t), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	var targets []struct {
		Type  string `json:"type"`
		Title string `json:"title"`
	}
	var listErr error
	defer func() {
		if t.Failed() {
			t.Logf("GET %s: targets %+v, last error: %v", url, targets, listErr)
		}
	}()
	wait.Until(t, fmt.Sprintf("a page target titled %q in %s", title, url), func() bool {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			listErr = err
			return false
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			listErr = fmt.Errorf("HTTP %s", res.Status)
			return false
		}
		targets = nil
		listErr = json.NewDecoder(res.Body).Decode(&targets)
		if listErr != nil {
			return false
		}
		for _, target := range targets {
			if target.Type == "page" && target.Title == title {
				return true
			}
		}
		return false
	})
}
