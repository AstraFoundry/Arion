package automationcli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

func TestParseArgs(t *testing.T) {
	spec := flagSpec{"settle": false, "timeout": true, "interactive": false, "mods": true}
	a, err := parseArgs([]string{"@e2", "--settle", "-i", "0", "-120", "--timeout=2s", "--mods", "shift,ctrl", "-.5", "-", "--", "--settle"}, spec)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.pos, []string{"@e2", "0", "-120", "-.5", "-", "--settle"}) || !a.bools["settle"] || !a.bools["interactive"] || a.vals["timeout"] != "2s" || a.vals["mods"] != "shift,ctrl" {
		t.Errorf("parsed %+v", a)
	}
	for _, bad := range [][]string{{"--nope"}, {"--timeout"}, {"--settle=1"}} {
		if _, err := parseArgs(bad, spec); err == nil {
			t.Errorf("%v parsed", bad)
		}
	}
}

// fakeApp answers the socket's requests as an app does, with a window of
// native UI (1) and one showing a page (2).
type fakeApp struct {
	mu    sync.Mutex
	nodes []map[string]any
	focus string
	calls []string
	// acted are the params of act and input requests.
	acted []map[string]any
	// idle can keep a request waiting while the app still draws.
	idle func()
	// trees are successive snapshots, as work changes what the app shows.
	trees [][]map[string]any
}

func (f *fakeApp) answer(method string, p map[string]any) (any, *protoError) {
	if method == "idle" && f.idle != nil {
		f.idle()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, method)
	window := 0
	if w, ok := p["window"].(float64); ok {
		window = int(w)
	}
	if window > 2 {
		return nil, &protoError{"no_window", "no such window"}
	}
	page := window == 2
	switch method {
	case "windows":
		return map[string]any{"windows": []map[string]any{
			{"id": 1, "title": "Contacts", "kind": "content", "focused": true, "visible": true},
			{"id": 2, "title": "Docs", "kind": "page", "url": "http://localhost/", "visible": true},
		}}, nil
	case "tree":
		if page {
			return nil, &protoError{"unsupported", "window 2 shows a web page: use eval"}
		}
		if len(f.trees) > 0 {
			f.nodes, f.trees = f.trees[0], f.trees[1:]
		}
		return map[string]any{"window": 1, "focus": f.focus, "nodes": f.nodes}, nil
	case "idle":
		return map[string]any{}, nil
	case "act":
		f.acted = append(f.acted, p)
		for _, n := range f.nodes {
			if n["id"] == p["id"] {
				if p["id"] == "11077037771207380721" && p["action"] == "press" && f.focus == "" {
					// Add shows a text field.
					f.nodes = append(f.nodes, map[string]any{"id": "42", "role": "textField", "label": "First name", "actions": []string{"focus", "setValue"}})
				}
				if p["action"] == "setValue" {
					n["value"] = p["text"]
					f.focus = n["id"].(string)
				}
				return map[string]any{}, nil
			}
		}
		return nil, &protoError{"no_element", "window 1 shows no element " + p["id"].(string)}
	case "input", "menu":
		if page && method == "input" {
			return nil, &protoError{"unsupported", "window 2 shows a web page: use eval"}
		}
		f.acted = append(f.acted, p)
		return map[string]any{}, nil
	case "eval":
		if !page {
			return nil, &protoError{"unsupported", "window 1 shows native UI, not a web page: use tree"}
		}
		if p["script"] == "document.title" {
			return map[string]any{"value": "Docs"}, nil
		}
		return map[string]any{"value": map[string]any{"n": 1}}, nil
	case "capture":
		return map[string]any{"png": []byte("\x89PNG fake"), "width": 2, "height": 1}, nil
	}
	return nil, &protoError{"unknown_method", method}
}

// serve answers the socket's connections until the test ends.
func (f *fakeApp) serve(t *testing.T) string {
	dir, err := os.MkdirTemp("", "mygo-automation")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "app.sock")
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				sc := bufio.NewScanner(c)
				for sc.Scan() {
					var req struct {
						ID     json.RawMessage
						Method string
						Params map[string]any
					}
					json.Unmarshal(sc.Bytes(), &req)
					res, perr := f.answer(req.Method, req.Params)
					line, _ := json.Marshal(map[string]any{"id": req.ID, "result": res, "error": perr})
					if perr != nil {
						line, _ = json.Marshal(map[string]any{"id": req.ID, "error": perr})
					}
					c.Write(append(line, '\n'))
				}
			}()
		}
	}()
	return path
}

// cli runs the CLI's command lines in one directory of sessions.
type cli struct {
	t   *testing.T
	dir string
}

// run runs a command line and returns its output and exit status.
func (c cli) run(args ...string) (string, int) {
	c.t.Helper()
	var out bytes.Buffer
	code := Run(args, strings.NewReader("1 + 1"), &out, &out, Options{Dir: c.dir, Skill: []byte("---\nname: mygo-automation\n---\nguide\n")})
	return out.String(), code
}

// must runs a command line that succeeds and checks its output.
func (c cli) must(want string, args ...string) {
	c.t.Helper()
	out, code := c.run(args...)
	if code != 0 || out != want {
		c.t.Errorf("mygo-automation %s: exit %d\n%s\nwant:\n%s", strings.Join(args, " "), code, out, want)
	}
}

// fails runs a command line that fails, with an output that has want.
func (c cli) fails(code int, want string, args ...string) {
	c.t.Helper()
	out, got := c.run(args...)
	if got != code || !strings.Contains(out, want) {
		c.t.Errorf("mygo-automation %s: exit %d (want %d)\n%s\nwant it to say: %s", strings.Join(args, " "), got, code, out, want)
	}
}

func TestCommands(t *testing.T) {
	app := &fakeApp{nodes: []map[string]any{
		{"id": "1", "role": "heading", "label": "Contacts"},
		{"id": "11077037771207380721", "role": "button", "key": "add", "label": "Add", "states": []string{"focusable"}, "actions": []string{"press", "focus"}},
		{"id": "3", "role": "list", "label": "People"},
		{"id": "4", "parent": "3", "role": "listItem", "label": "Ada", "key": 7, "actions": []string{"press"}, "bounds": map[string]any{"x": 10, "y": 20, "width": 100, "height": 30}},
	}}
	path := app.serve(t)
	c := cli{t, t.TempDir()}

	c.fails(1, "no session \"default\": run mygo-automation open", "snapshot")
	c.fails(2, "open needs --socket", "open")
	c.must("opened "+path+" (session default)\nw1 [content] \"Contacts\" focused\nw2 [page] \"Docs\" url=http://localhost/\n", "open", "--socket", path)
	c.must("@e2 [button] \"Add\" key=add\n@e4 [listItem] \"Ada\" key=7\n", "snapshot", "-i")
	c.must("@e1 [heading] \"Contacts\"\n@e2 [button] \"Add\" key=add\n@e3 [list] \"People\"\n  @e4 [listItem] \"Ada\" key=7\n", "snapshot")

	// Acting, then what changed once the window settled, with the
	// elements around the change and the new element's ref.
	c.must("= @e4 [listItem] \"Ada\" key=7\n+ @e5 [textField] \"First name\"\n", "press", "@e2", "--settle")
	c.must("= @e4 [listItem] \"Ada\" key=7\n- @e5 [textField] \"First name\"\n+ @e5 [textField] \"First name\" value=\"Ada\" focused\n", "fill", "@e5", "Ada", "--settle")
	c.must("(no changes)\n", "diff")
	c.must("ok\n", "press", "key=add")
	if got := app.acted[len(app.acted)-1]; got["id"] != "11077037771207380721" || got["window"] != float64(1) {
		t.Errorf("press key=add sent %v", got)
	}
	c.must("ok\n", "press", "key=7")
	c.must("ok\n", "act", "@e5", "selectText", "0", "2")
	if got := app.acted[len(app.acted)-1]; got["action"] != "selectText" || got["from"] != float64(0) || got["to"] != float64(2) {
		t.Errorf("act selectText sent %v", got)
	}

	// Pointers go to an element's center, or to a point.
	c.must("ok\n", "click", "@e4", "--button", "right", "--mods", "shift")
	c.must("ok\n", "wheel", "@e4", "0", "-120", "--precise")
	c.must("ok\n", "pointer", "down", "5", "6.5")
	c.must("ok\n", "key", "CmdOrCtrl+A", "Backspace")
	c.must("ok\n", "type", "Grace")
	c.must("ok\n", "menu", "File", "New Window")
	var events []any
	for _, p := range app.acted[len(app.acted)-6:] {
		if e, ok := p["events"].([]any); ok {
			events = append(events, e...)
		} else {
			events = append(events, p["path"])
		}
	}
	got, _ := json.Marshal(events)
	want := `[{"button":"right","mods":["shift"],"type":"click","x":60,"y":35},{"dy":-120,"precise":true,"type":"wheel","x":60,"y":35},{"type":"pointerDown","x":5,"y":6.5},{"key":"CmdOrCtrl+A","type":"key"},{"key":"Backspace","type":"key"},{"text":"Grace","type":"text"},["File","New Window"]]`
	if string(got) != want {
		t.Errorf("events\n got %s\nwant %s", got, want)
	}

	// A ref whose element is gone is rejected, never given to another.
	app.mu.Lock()
	app.nodes = slices.Delete(app.nodes, 3, 4)
	app.mu.Unlock()
	c.fails(1, "stale ref @e4: window 1 no longer shows its element", "press", "@e4")
	c.fails(1, "stale ref @e4: window 1 no longer shows its element", "click", "@e4")
	c.must("= @e3 [list] \"People\"\n- @e4 [listItem] \"Ada\" key=7\n= @e5 [textField] \"First name\" value=\"Ada\" focused\n", "diff")
	c.fails(1, "stale ref @e4: its element is gone", "press", "@e4")
	c.fails(1, "stale ref @e9: no snapshot or diff", "press", "@e9")
	c.fails(1, "no_element: window 1 shows no element with the key 7", "press", "key=7")
	c.fails(2, `selector "Add"`, "press", "Add")

	// Pages are read with eval; native UI is not.
	c.must("w2 [page] \"Docs\" url=http://localhost/ selected\n", "window", "w2")
	c.must("w1 [content] \"Contacts\" focused\nw2 [page] \"Docs\" url=http://localhost/ selected\n", "windows")
	c.fails(1, "window 2 shows a web page, which has no elements here: read it with eval", "snapshot")
	c.must("Docs\n", "eval", "document.title")
	c.must("{\"n\":1}\n", "eval", "-")
	c.must("\"Docs\"\n", "eval", "document.title", "--json")
	c.must("ok (a web page: read it with eval)\n", "wait", "--settle")
	c.fails(1, "window 1 shows native UI, not a web page: read it with snapshot", "eval", "1", "--window", "w1")
	c.must("commands target the focused window\n", "window", "auto")

	shot := filepath.Join(t.TempDir(), "shot.png")
	c.must(shot+" 2x1\n", "screenshot", shot)
	if b, err := os.ReadFile(shot); err != nil || string(b) != "\x89PNG fake" {
		t.Errorf("the screenshot's file: %q, %v", b, err)
	}

	// Sessions keep refs of their own.
	c.must("opened "+path+" (session b)\nw1 [content] \"Contacts\" focused\nw2 [page] \"Docs\" url=http://localhost/\n", "open", "--socket", path, "--session", "b")
	c.must("@e1 [heading] \"Contacts\"\n@e2 [button] \"Add\" key=add\n@e3 [list] \"People\"\n@e4 [textField] \"First name\" value=\"Ada\" focused\n", "snapshot", "--session", "b")
	c.fails(1, "stale ref @e4", "press", "@e4")

	// Opening a session again makes its refs stale, never names others
	// with them.
	c.must("opened "+path+" (session b)\nw1 [content] \"Contacts\" focused\nw2 [page] \"Docs\" url=http://localhost/\n", "open", "--socket", path, "--session", "b")
	c.fails(1, "stale ref @e1: its element is gone", "press", "@e1", "--session", "b")
	c.must("@e6 [button] \"Add\" key=add\n@e8 [textField] \"First name\" value=\"Ada\" focused\n", "snapshot", "-i", "--session", "b")

	c.must("---\nname: mygo-automation\n---\nguide\n", "skills")
	c.must("closed session default\n", "close")
	c.must("no session default\n", "close")
	c.fails(1, "no session \"default\"", "snapshot")
	c.fails(2, "no command \"nope\"", "nope")
}

func TestWaitText(t *testing.T) {
	app := &fakeApp{nodes: []map[string]any{{"id": "1", "role": "text", "label": "Saving…"}}}
	path := app.serve(t)
	c := cli{t, t.TempDir()}
	c.must("opened "+path+" (session default)\nw1 [content] \"Contacts\" focused\nw2 [page] \"Docs\" url=http://localhost/\n", "open", "--socket", path)
	c.must("ok\n", "wait", "--text", "Saving")
	c.fails(1, `timeout: window 1 does not show text "Saved" after 1ms`, "wait", "--text", "Saved", "--timeout", "1ms")
	c.fails(1, `timeout: window 1 still shows text "Saving" after 1ms`, "wait", "--text", "Saving", "--gone", "--timeout", "1")
	c.fails(2, "wait --gone needs", "wait", "--gone")
}

// A condition can be met while a window keeps drawing. Waiting for it
// must not need that drawing to stop, even with no timeout.
func TestWaitWhileDrawing(t *testing.T) {
	ready := []map[string]any{{"id": "1", "role": "text", "label": "Ready", "key": "ready"}}
	saving := []map[string]any{{"id": "1", "role": "text", "label": "Saving"}}
	value := []map[string]any{{"id": "1", "role": "textField", "value": "Ready"}}
	for _, tc := range []struct {
		name     string
		args     []string
		trees    [][]map[string]any
		snapshot bool
		code     int
		out      string
	}{
		{"visible", []string{"--text", "Ready"}, [][]map[string]any{ready}, false, 0, "ok\n"},
		{"no-timeout", []string{"--text", "Ready", "--timeout", "0"}, [][]map[string]any{ready}, false, 0, "ok\n"},
		{"later", []string{"--text", "Ready", "--timeout", "0"}, [][]map[string]any{saving, ready}, false, 0, "ok\n"},
		{"value", []string{"--text", "Ready", "--timeout", "0"}, [][]map[string]any{saving, value}, false, 0, "ok\n"},
		{"key", []string{"key=ready", "--timeout", "0"}, [][]map[string]any{nil, ready}, false, 0, "ok\n"},
		{"gone-text", []string{"--text", "Saving", "--gone", "--timeout", "0"}, [][]map[string]any{saving, ready}, false, 0, "ok\n"},
		{"gone-id", []string{"id=1", "--gone", "--timeout", "0"}, [][]map[string]any{ready, nil}, false, 0, "ok\n"},
		{"gone-ref", []string{"@e1", "--gone", "--timeout", "0"}, [][]map[string]any{ready, ready, nil}, true, 0, "ok\n"},
		{"timeout", []string{"--text", "Ready", "--timeout", "1ms"}, [][]map[string]any{saving}, false, 1, "error: timeout: window 1 does not show text \"Ready\" after 1ms\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := &fakeApp{trees: tc.trees}
			idle := make(chan struct{}, 1)
			stopDrawing := make(chan struct{})
			stop := sync.OnceFunc(func() { close(stopDrawing) })
			defer stop()
			app.idle = func() {
				select {
				case idle <- struct{}{}:
				default:
				}
				<-stopDrawing
			}
			path := app.serve(t)
			c := cli{t, t.TempDir()}
			if out, code := c.run("open", "--socket", path); code != 0 {
				t.Fatalf("open: exit %d: %s", code, out)
			}
			if tc.snapshot {
				c.must("@e1 [text] \"Ready\" key=ready\n", "snapshot")
			}
			type result struct {
				out  string
				code int
			}
			done := make(chan result, 1)
			go func() {
				out, code := c.run(append([]string{"wait"}, tc.args...)...)
				done <- result{out, code}
			}()
			var got result
			select {
			case got = <-done:
			case <-idle:
				t.Error("condition wait needs the window to stop drawing")
				stop()
				got = wait.For(t, "the wait command to finish", done)
			case <-wait.Context(t).Done():
				t.Fatal("the wait command never finished or asked the app to settle")
			}
			if got.code != tc.code || got.out != tc.out {
				t.Errorf("wait: exit %d: %s; want exit %d: %s", got.code, got.out, tc.code, tc.out)
			}
		})
	}
}

func TestWaitSettles(t *testing.T) {
	for _, condition := range []bool{false, true} {
		app := &fakeApp{nodes: []map[string]any{{"id": "1", "role": "text", "label": "Ready"}}}
		app.idle = func() {
			app.mu.Lock()
			defer app.mu.Unlock()
			app.nodes = []map[string]any{{"id": "1", "role": "text", "label": "Settled"}}
		}
		path := app.serve(t)
		c := cli{t, t.TempDir()}
		if out, code := c.run("open", "--socket", path); code != 0 {
			t.Fatalf("open: exit %d: %s", code, out)
		}
		if condition {
			c.must("+ @e1 [text] \"Settled\"\n", "wait", "--text", "Ready", "--settle")
		} else {
			c.must("ok\n", "wait")
			c.must("@e1 [text] \"Settled\"\n", "snapshot")
		}
	}
}
