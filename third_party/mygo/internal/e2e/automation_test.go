//go:build !mygo_noinspector && !js

package e2e

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	_ "github.com/egoist/mygo/plugins/automation"
	"github.com/egoist/mygo/ui"
)

// automationApp is the app TestAutomation and TestAutomationCLI drive
// from another process: a counter, a text field of native UI and a
// greeting shown while the field has text, and a menu that resets the
// counter and quits. MYGO_AUTOMATION names its socket.
func automationApp() {
	count, name := 0, ""
	var w *mygo.Window
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Label: "Test", Submenu: []*mygo.MenuItem{
		{Label: "Reset", Click: func(*mygo.MenuItem, *mygo.Window) {
			count = 0
			w.Invalidate()
		}},
		{Label: "Quit", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
	}}}))
	view := func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			ui.Text(c, fmt.Sprintf("Count: %d", count))
			if ui.Button(c, "Add").Key("add").Clicked() {
				count++
			}
			c.Key("name")
			ui.TextInput(c, &name).Label("Name")
			if name != "" {
				ui.Text(c, "Hello, "+name)
			}
		})
	}
	mygo.App.WhenReady(func() {
		w = mygo.NewWindow(mygo.WindowOptions{Title: "Automation", Width: 400, Height: 300, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// automationClient sends requests to an app's socket, one at a time.
type automationClient struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
	n    int
}

type automationError struct{ Code, Message string }

// call sends a request and returns its result, or its error.
func (c *automationClient) call(method string, params any) (json.RawMessage, *automationError) {
	c.t.Helper()
	c.n++
	line, err := json.Marshal(map[string]any{"id": c.n, "method": method, "params": params})
	if err != nil {
		c.t.Fatal(err)
	}
	if _, err := c.conn.Write(append(line, '\n')); err != nil {
		c.t.Fatal(err)
	}
	answer, err := c.r.ReadBytes('\n')
	if err != nil {
		c.t.Fatalf("%s: %v", method, err)
	}
	var res struct {
		ID     int
		Result json.RawMessage
		Error  *automationError
	}
	if err := json.Unmarshal(answer, &res); err != nil || res.ID != c.n {
		c.t.Fatalf("%s: answer %s (%v)", method, answer, err)
	}
	return res.Result, res.Error
}

// must is call for requests that succeed, decoding the result into v.
func (c *automationClient) must(method string, params any, v any) {
	c.t.Helper()
	res, e := c.call(method, params)
	if e != nil {
		c.t.Fatalf("%s %v: %s: %s", method, params, e.Code, e.Message)
	}
	if v != nil {
		if err := json.Unmarshal(res, v); err != nil {
			c.t.Fatalf("%s: %s: %v", method, res, err)
		}
	}
}

type automationNode struct {
	ID, Role, Label, Value string
	Key                    any
	Bounds                 struct{ X, Y, Width, Height float64 }
}

// automationProc is automationApp running in a process of its own.
type automationProc struct {
	path   string
	exited chan error
	// ended is set once exited was read.
	ended bool
}

// startAutomationApp starts automationApp listening on a socket of its
// own, and stops it when the test ends.
func startAutomationApp(t *testing.T) *automationProc {
	t.Helper()
	dir, err := os.MkdirTemp("", "mygo-e2e")
	if err != nil {
		t.Fatal(err)
	}
	p := &automationProc{path: filepath.Join(dir, "app.sock"), exited: make(chan error, 1)}
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "MYGO_E2E_AUTOMATION_APP=1", "MYGO_AUTOMATION="+p.path)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { p.exited <- cmd.Wait() }()
	t.Cleanup(func() {
		if !p.ended {
			cmd.Process.Kill()
			<-p.exited
		}
		if t.Failed() {
			t.Logf("the app's output:\n%s", out.String())
		}
		os.RemoveAll(dir)
	})
	return p
}

// dial connects to the app's socket once it listens.
func (p *automationProc) dial(t *testing.T) net.Conn {
	t.Helper()
	var conn net.Conn
	wait.Until(t, "the app's socket", func() bool {
		select {
		case err := <-p.exited:
			p.ended = true
			t.Fatalf("the app exited: %v", err)
		default:
		}
		var err error
		conn, err = net.Dial("unix", p.path)
		return err == nil
	})
	return conn
}

// TestAutomation drives an app of native UI from another process through
// plugins/automation's socket: it lists the windows, reads the elements,
// presses a button, clicks a text field and types, chooses menu items,
// captures the window and waits for it to settle.
func TestAutomation(t *testing.T) {
	app := startAutomationApp(t)
	path := app.path
	conn := app.dial(t)
	defer conn.Close()
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("the socket's mode: %v, %v", fi.Mode(), err)
		}
	}
	c := &automationClient{t: t, conn: conn, r: bufio.NewReader(conn)}

	type window struct {
		ID          int
		Title, Kind string
	}
	var win window
	wait.Until(t, "the app's window", func() bool {
		var res struct{ Windows []window }
		c.must("windows", nil, &res)
		if len(res.Windows) == 1 {
			win = res.Windows[0]
		}
		return len(res.Windows) == 1
	})
	if win.Title != "Automation" || win.Kind != "content" {
		t.Fatalf("window %+v", win)
	}
	at := map[string]any{"window": win.ID}
	c.must("idle", at, nil)

	tree := func() []automationNode {
		var res struct{ Nodes []automationNode }
		c.must("tree", at, &res)
		return res.Nodes
	}
	find := func(role, label string) automationNode {
		t.Helper()
		nodes := tree()
		for _, n := range nodes {
			if n.Role == role && n.Label == label {
				return n
			}
		}
		t.Fatalf("no %s %q in %+v", role, label, nodes)
		return automationNode{}
	}
	find("text", "Count: 0")
	add := find("button", "Add")
	if add.Key != "add" || add.ID == "" || add.Bounds.Width == 0 {
		t.Errorf("the button %+v", add)
	}
	field := find("textField", "Name")
	if field.Key != "name" {
		t.Errorf("the text field %+v", field)
	}

	// Pressing as assistive technology does, by the key and by the ID.
	c.must("act", map[string]any{"window": win.ID, "key": "add", "action": "press"}, nil)
	c.must("idle", at, nil)
	find("text", "Count: 1")
	c.must("act", map[string]any{"window": win.ID, "id": add.ID, "action": "press"}, nil)
	c.must("idle", at, nil)
	find("text", "Count: 2")

	// Clicking the text field, then typing into it.
	b := field.Bounds
	c.must("input", map[string]any{"window": win.ID, "events": []any{
		map[string]any{"type": "click", "x": b.X + b.Width/2, "y": b.Y + b.Height/2},
	}}, nil)
	c.must("idle", at, nil)
	c.must("input", map[string]any{"window": win.ID, "events": []any{map[string]any{"type": "text", "text": "Grace"}}}, nil)
	c.must("idle", at, nil)
	find("text", "Hello, Grace")
	c.must("input", map[string]any{"window": win.ID, "events": []any{map[string]any{"type": "key", "key": "Backspace"}}}, nil)
	c.must("idle", at, nil)
	if f := find("textField", "Name"); f.Value != "Grac" {
		t.Errorf("the text field after Backspace: %+v", f)
	}

	// The window as an image.
	var shot struct {
		PNG           string
		Width, Height int
	}
	c.must("capture", at, &shot)
	data, err := base64.StdEncoding.DecodeString(shot.PNG)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if s := img.Bounds().Size(); s.X != shot.Width || s.Y != shot.Height || s.X < 200 || s.Y < 150 {
		t.Errorf("the capture is %v, said %dx%d", s, shot.Width, shot.Height)
	}

	// What the window cannot do, and what no window can.
	if _, e := c.call("eval", map[string]any{"window": win.ID, "script": "1"}); e == nil || e.Code != "unsupported" {
		t.Errorf("eval in native UI: %+v", e)
	}
	if _, e := c.call("tree", map[string]any{"window": win.ID + 1000}); e == nil || e.Code != "no_window" {
		t.Errorf("tree of no window: %+v", e)
	}
	if _, e := c.call("act", map[string]any{"window": win.ID, "id": "1", "action": "press"}); e == nil || e.Code != "no_element" {
		t.Errorf("act on no element: %+v", e)
	}

	// The application menu.
	if _, e := c.call("menu", map[string]any{"window": win.ID, "path": []string{"Test", "Reset"}}); e != nil {
		t.Skipf("menus cannot be chosen on this platform: %s: %s", e.Code, e.Message)
	}
	c.must("idle", at, nil)
	find("text", "Count: 0")
	// Quitting closes the socket, which leaves no file. The answer may
	// not come.
	line, _ := json.Marshal(map[string]any{"id": 0, "method": "menu", "params": map[string]any{"window": win.ID, "path": []string{"Test", "Quit"}}})
	conn.Write(append(line, '\n'))
	if err := wait.For(t, "the app to quit", app.exited); err != nil {
		t.Errorf("the app exited: %v", err)
	}
	app.ended = true
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("the socket stays: %v", err)
	}
}
