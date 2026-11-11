//go:build windows && (amd64 || arm64)

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// TestCookieViewReleased makes cookie calls in an app showing native UI
// only, whose calls go through a hidden web view: once none needs it, the
// web view closes and WebView2's processes exit, unless it alone keeps a
// session cookie (#1109). The helper runs in a user data folder of its
// own, so that the browser process is its child, not that of the test
// process's pages.
func TestCookieViewReleased(t *testing.T) {
	// Not t.TempDir: the browser processes outlive the helper for a
	// moment and keep files of the folder open.
	folder, err := os.MkdirTemp("", "mygo-cookie-view-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		wait.Until(t, "WebView2's browser processes to let go of "+folder, func() bool { return os.RemoveAll(folder) == nil })
	})
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "MYGO_E2E_COOKIE_VIEW=1", "WEBVIEW2_USER_DATA_FOLDER="+folder)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "cookie view: released\n") {
		t.Errorf("exit: %v, output %q", err, out)
	}
}

// cookieViewApp is the helper process of TestCookieViewReleased.
func cookieViewApp() {
	report := func(format string, args ...any) { fmt.Printf("cookie view: "+format+"\n", args...) }
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Cookie view", Width: 200, Height: 100,
			Content: ui.View(func(c *ui.Context) { ui.Text(c, "Native UI") })})
		go func() {
			defer mygo.App.Quit()
			if _, err := mygo.Cookies.List(mygo.CookieFilter{URL: "https://example.com/"}); err != nil {
				report("List: %v", err)
				return
			}
			if !wait.Poll(func() bool { return len(webViewProcesses()) == 0 }) {
				report("WebView2's processes still run after List: %v", webViewProcesses())
				return
			}
			// A session cookie lives as long as WebView2's processes: the
			// hidden web view stays. Checking that nothing happens may
			// wait a while: longer than the backend keeps an idle hidden
			// web view before it checks (cookieViewIdle), which a slow
			// machine only makes this miss.
			c := mygo.Cookie{Name: fmt.Sprintf("mygo%dsession", time.Now().UnixNano()), Value: "kept", Domain: "mygo-cookie-view.test"}
			if err := mygo.Cookies.Set(c); err != nil {
				report("Set: %v", err)
				return
			}
			time.Sleep(8 * time.Second)
			if list, err := mygo.Cookies.List(mygo.CookieFilter{Name: c.Name}); err != nil || len(list) != 1 {
				report("the session cookie went: %+v %v", list, err)
				return
			}
			if err := mygo.Cookies.Delete(c); err != nil {
				report("Delete: %v", err)
				return
			}
			if !wait.Poll(func() bool { return len(webViewProcesses()) == 0 }) {
				report("WebView2's processes still run after Delete: %v", webViewProcesses())
				return
			}
			report("released")
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// webViewProcesses returns the WebView2 processes this process started,
// and theirs.
func webViewProcesses() []int {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(snap)
	children := map[int][]int{}
	names := map[int]string{}
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err := syscall.Process32First(snap, &e); err == nil; err = syscall.Process32Next(snap, &e) {
		children[int(e.ParentProcessID)] = append(children[int(e.ParentProcessID)], int(e.ProcessID))
		names[int(e.ProcessID)] = syscall.UTF16ToString(e.ExeFile[:])
	}
	var out []int
	seen := map[int]bool{os.Getpid(): true}
	for queue := slices.Clone(children[os.Getpid()]); len(queue) > 0; queue = queue[1:] {
		p := queue[0]
		if seen[p] {
			continue
		}
		seen[p] = true
		if strings.EqualFold(names[p], "msedgewebview2.exe") {
			out = append(out, p)
		}
		queue = append(queue, children[p]...)
	}
	return out
}
