//go:build !mygo_noinspector

package automation

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// fakeDriver records what requests ask of the app.
type fakeDriver struct {
	access *platform.AccessTree
	keys   map[uint64]any
	acted  []platform.SurfaceEvent
	sent   []platform.SurfaceEvent
	menus  [][]string
	png    []byte
	evalFn func(string) (any, error)
	waited time.Duration
}

func (d *fakeDriver) windows() ([]windowInfo, error) {
	return []windowInfo{{ID: 1, Title: "Counter", Kind: "content", Bounds: rect{10, 20, 300, 200}, Focused: true, Visible: true}}, nil
}

func (d *fakeDriver) tree(window int) (int, *platform.AccessTree, map[uint64]any, error) {
	if window > 1 {
		return 0, nil, nil, errorf(codeNoWindow, "no window %d", window)
	}
	return 1, d.access, d.keys, nil
}

func (d *fakeDriver) act(window int, ev platform.SurfaceEvent) error {
	d.acted = append(d.acted, ev)
	return nil
}

func (d *fakeDriver) input(window int, evs []platform.SurfaceEvent) error {
	d.sent = append(d.sent, evs...)
	return nil
}

func (d *fakeDriver) menu(window int, path []string) error {
	d.menus = append(d.menus, path)
	return nil
}

func (d *fakeDriver) capture(window int) ([]byte, error) { return d.png, nil }

func (d *fakeDriver) eval(window int, script string) (any, error) { return d.evalFn(script) }

func (d *fakeDriver) idle(window int, timeout time.Duration) error {
	d.waited = timeout
	return nil
}

// do answers line, as JSON.
func do(t *testing.T, d driver, line string) map[string]any {
	t.Helper()
	b, err := json.Marshal(handle(d, []byte(line)))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// errCode returns the code of the error of a response, "" for none.
func errCode(m map[string]any) string {
	e, _ := m["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func TestRequestErrors(t *testing.T) {
	d := &fakeDriver{}
	for _, c := range []struct{ line, code string }{
		{`not json`, codeParse},
		{`[1, 2]`, codeParse},
		{`{"id": 1}`, codeInvalid},
		{`{"id": 1, "method": "fly"}`, codeUnknownMethod},
		{`{"id": 1, "method": "tree", "params": {"window": 1, "windwo": 2}}`, codeInvalid},
		{`{"id": 1, "method": "tree", "params": {"window": "one"}}`, codeInvalid},
		{`{"id": 1, "method": "tree", "params": {"window": 7}}`, codeNoWindow},
		{`{"id": 1, "method": "windows", "params": {"window": 1}}`, codeInvalid},
		{`{"id": 1, "method": "menu", "params": {"path": []}}`, codeInvalid},
		{`{"id": 1, "method": "idle", "params": {"timeout": -1}}`, codeInvalid},
	} {
		m := do(t, d, c.line)
		if got := errCode(m); got != c.code {
			t.Errorf("%s: error %v, want %s", c.line, m["error"], c.code)
		}
		if _, ok := m["result"]; ok {
			t.Errorf("%s: a result with the error: %v", c.line, m)
		}
	}
	// The ID comes back as it went, or null when the line has none.
	if m := do(t, d, `{"id": "a-7", "method": "fly"}`); m["id"] != "a-7" {
		t.Errorf("id %v", m["id"])
	}
	if b, _ := json.Marshal(handle(d, []byte("{"))); !bytes.Contains(b, []byte(`"id":null`)) {
		t.Errorf("an unreadable line's answer: %s", b)
	}
}

func TestWindowsRequest(t *testing.T) {
	m := do(t, &fakeDriver{}, `{"id": 1, "method": "windows"}`)
	ws := m["result"].(map[string]any)["windows"].([]any)
	w := ws[0].(map[string]any)
	if w["id"] != 1.0 || w["kind"] != "content" || w["focused"] != true || w["bounds"].(map[string]any)["width"] != 300.0 {
		t.Errorf("windows: %v", ws)
	}
}

// testTree is a window's tree: a button in a group, and a text field.
func testTree() (*platform.AccessTree, map[uint64]any) {
	t := &platform.AccessTree{Focus: 1 << 60, Nodes: []platform.AccessNode{
		{ID: 7, Parent: -1, Role: platform.RoleGroup},
		{ID: 1 << 60, Parent: 0, Role: platform.RoleButton, Label: "Add", KeyShortcuts: "Ctrl+N",
			States: platform.AccessFocusable, Actions: platform.ActionPress | platform.ActionFocus,
			Bounds: platform.RectF{X: 1, Y: 2, W: 30, H: 20}},
		{ID: 9, Parent: 0, Role: platform.RoleTextField, Value: "hey", SelStart: 3, SelEnd: 3, AutoComplete: platform.AutoCompleteList},
		{ID: 11, Parent: 0, Role: platform.RoleSlider, Min: 0, Max: 10, Now: 4},
	}}
	return t, map[uint64]any{1 << 60: "add", 9: 42}
}

func TestTreeRequest(t *testing.T) {
	tree, keys := testTree()
	m := do(t, &fakeDriver{access: tree, keys: keys}, `{"id": 1, "method": "tree"}`)
	r := m["result"].(map[string]any)
	if r["window"] != 1.0 || r["focus"] != "1152921504606846976" {
		t.Errorf("tree: %v", r)
	}
	nodes := r["nodes"].([]any)
	group, button, field, slider := nodes[0].(map[string]any), nodes[1].(map[string]any), nodes[2].(map[string]any), nodes[3].(map[string]any)
	if _, ok := group["parent"]; ok || group["role"] != "group" {
		t.Errorf("group: %v", group)
	}
	// IDs go as strings, which JavaScript's numbers would round.
	if button["id"] != "1152921504606846976" || button["parent"] != "7" || button["role"] != "button" || button["key"] != "add" || button["label"] != "Add" || button["keyShortcuts"] != "Ctrl+N" {
		t.Errorf("button: %v", button)
	}
	if !slices.Equal(button["states"].([]any), []any{"focusable"}) || !slices.Equal(button["actions"].([]any), []any{"press", "focus"}) {
		t.Errorf("button's states and actions: %v %v", button["states"], button["actions"])
	}
	if b := button["bounds"].(map[string]any); b["x"] != 1.0 || b["height"] != 20.0 {
		t.Errorf("button's bounds: %v", b)
	}
	if field["key"] != 42.0 || field["value"] != "hey" || !slices.Equal(field["selection"].([]any), []any{3.0, 3.0}) || field["autoComplete"] != "list" {
		t.Errorf("text field: %v", field)
	}
	if slider["max"] != 10.0 || slider["now"] != 4.0 || slider["min"] != 0.0 {
		t.Errorf("slider: %v", slider)
	}
	if _, ok := field["keyShortcuts"]; ok {
		t.Errorf("a text field with key shortcuts: %v", field)
	}
	if _, ok := button["autoComplete"]; ok {
		t.Errorf("a button with suggestions: %v", button)
	}
	if _, ok := button["min"]; ok {
		t.Errorf("a button with a range: %v", button)
	}
}

// A grid gives its size and its rows and cells their places, as the app
// gave them, and the columns a cell covers, and says it chooses several
// rows.
func TestTreeGrid(t *testing.T) {
	tree := &platform.AccessTree{Nodes: []platform.AccessNode{
		{ID: 1, Parent: -1, Role: platform.RoleGrid, Label: "Sheet", RowCount: 10001, ColCount: 5, States: platform.AccessSelectable | platform.AccessMultiselectable},
		{ID: 2, Parent: 0, Role: platform.RoleRow, RowIndex: 4122},
		{ID: 3, Parent: 1, Role: platform.RoleCell, Label: "B4121", ColIndex: 2, ColSpan: 2},
	}}
	m := do(t, &fakeDriver{access: tree}, `{"id": 1, "method": "tree"}`)
	nodes := m["result"].(map[string]any)["nodes"].([]any)
	grid, row, cell := nodes[0].(map[string]any), nodes[1].(map[string]any), nodes[2].(map[string]any)
	if grid["role"] != "grid" || grid["rowCount"] != 10001.0 || grid["colCount"] != 5.0 || !slices.Contains(grid["states"].([]any), "multiselectable") {
		t.Errorf("grid: %v", grid)
	}
	if row["rowIndex"] != 4122.0 || cell["colIndex"] != 2.0 || cell["colSpan"] != 2.0 {
		t.Errorf("row %v, cell %v", row, cell)
	}
	if _, ok := cell["rowIndex"]; ok {
		t.Errorf("a cell with a row index of its own: %v", cell)
	}
	if _, ok := cell["rowSpan"]; ok {
		t.Errorf("a cell covering one row with a row span: %v", cell)
	}
}

func TestActRequest(t *testing.T) {
	tree, keys := testTree()
	d := &fakeDriver{access: tree, keys: keys}
	for _, line := range []string{
		`{"id": 1, "method": "act", "params": {"id": "1152921504606846976", "action": "press"}}`,
		`{"id": 2, "method": "act", "params": {"key": "add", "action": "press"}}`,
		`{"id": 3, "method": "act", "params": {"key": 42, "action": "setValue", "text": "hi"}}`,
		`{"id": 4, "method": "act", "params": {"id": 9, "action": "selectText", "from": 1, "to": 2}}`,
	} {
		if m := do(t, d, line); m["error"] != nil {
			t.Fatalf("%s: %v", line, m["error"])
		}
	}
	want := []platform.SurfaceEvent{
		{Kind: platform.AccessAction, ID: 1 << 60, Action: platform.AccessPress},
		{Kind: platform.AccessAction, ID: 1 << 60, Action: platform.AccessPress},
		{Kind: platform.AccessAction, ID: 9, Action: platform.AccessSetValue, Text: "hi"},
		{Kind: platform.AccessAction, ID: 9, Action: platform.AccessSelectText, From: 1, To: 2},
	}
	if len(d.acted) != len(want) {
		t.Fatalf("acted %+v", d.acted)
	}
	for i := range want {
		if a, w := d.acted[i], want[i]; a.Kind != w.Kind || a.ID != w.ID || a.Action != w.Action || a.Text != w.Text || a.From != w.From || a.To != w.To {
			t.Errorf("action %d: %+v, want %+v", i, a, w)
		}
	}

	keys[11] = "add" // two elements have it now
	for _, c := range []struct{ line, code string }{
		{`{"id": 1, "method": "act", "params": {"action": "press"}}`, codeInvalid},
		{`{"id": 1, "method": "act", "params": {"id": "7", "key": "add", "action": "press"}}`, codeInvalid},
		{`{"id": 1, "method": "act", "params": {"id": "7", "action": "smash"}}`, codeInvalid},
		{`{"id": 1, "method": "act", "params": {"id": "seven", "action": "press"}}`, codeInvalid},
		{`{"id": 1, "method": "act", "params": {"key": "remove", "action": "press"}}`, codeNoElement},
		{`{"id": 1, "method": "act", "params": {"key": "add", "action": "press"}}`, codeInvalid},
	} {
		if m := do(t, d, c.line); errCode(m) != c.code {
			t.Errorf("%s: error %v, want %s", c.line, m["error"], c.code)
		}
	}
}

func TestInputRequest(t *testing.T) {
	d := &fakeDriver{}
	m := do(t, d, `{"id": 1, "method": "input", "params": {"events": [
		{"type": "click", "x": 5, "y": 6, "clicks": 2},
		{"type": "pointerDown", "x": 1, "y": 2, "button": "right", "mods": ["shift"]},
		{"type": "wheel", "x": 1, "y": 2, "dy": 40, "precise": true},
		{"type": "key", "key": "Ctrl+Shift+Z"},
		{"type": "keyDown", "key": "Enter"},
		{"type": "keyUp", "key": "F5"},
		{"type": "text", "text": "héllo"}
	]}}`)
	if m["error"] != nil {
		t.Fatal(m["error"])
	}
	type ev struct {
		kind          platform.SurfaceEventKind
		x, dy         float64
		button, click int
		key           platform.Key
		mods          platform.Modifiers
		text          string
	}
	var got []ev
	for _, e := range d.sent {
		got = append(got, ev{e.Kind, e.X, e.DY, e.Button, e.Clicks, e.Key, e.Mods, e.Text})
	}
	want := []ev{
		{kind: platform.PointerMove, x: 5},
		{kind: platform.PointerDown, x: 5, click: 1}, {kind: platform.PointerUp, x: 5, click: 1},
		{kind: platform.PointerDown, x: 5, click: 2}, {kind: platform.PointerUp, x: 5, click: 2},
		{kind: platform.PointerDown, x: 1, button: 1, click: 1, mods: platform.ModShift},
		{kind: platform.PointerScroll, x: 1, dy: 40},
		{kind: platform.KeyPressed, key: platform.KeyZ, mods: platform.ModCtrl | platform.ModShift},
		{kind: platform.KeyReleased, key: platform.KeyZ, mods: platform.ModCtrl | platform.ModShift},
		{kind: platform.KeyPressed, key: platform.KeyEnter},
		{kind: platform.KeyReleased, key: platform.KeyF5},
		{kind: platform.TextInput, text: "héllo"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("events:\n%+v\nwant\n%+v", got, want)
	}
	if !d.sent[6].Precise || d.sent[7].PhysicalKey != platform.KeyZ {
		t.Errorf("wheel %+v, key %+v", d.sent[6], d.sent[7])
	}

	for _, events := range []string{
		`[{"type": "hover"}]`,
		`[{"type": "key", "key": "Hyper+A"}]`,
		`[{"type": "key", "key": "VolumeUp"}]`,
		`[{"type": "click", "button": "fourth"}]`,
		`[{"type": "click", "mods": ["fn"]}]`,
		`[{"type": "text"}]`,
	} {
		n := len(d.sent)
		if m := do(t, d, `{"id": 1, "method": "input", "params": {"events": `+events+`}}`); errCode(m) != codeInvalid {
			t.Errorf("%s: error %v", events, m["error"])
		}
		if len(d.sent) != n {
			t.Errorf("%s: sent events before failing", events)
		}
	}
}

func TestMenuCaptureEvalIdle(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3, 2)))
	d := &fakeDriver{png: buf.Bytes(), evalFn: func(s string) (any, error) {
		if s == "throw 1" {
			return nil, errors.New("Uncaught 1")
		}
		return map[string]any{"title": "E2E"}, nil
	}}
	if m := do(t, d, `{"id": 1, "method": "menu", "params": {"path": ["File", "New"]}}`); m["error"] != nil || !slices.Equal(d.menus[0], []string{"File", "New"}) {
		t.Errorf("menu: %v %v", m, d.menus)
	}
	m := do(t, d, `{"id": 1, "method": "capture"}`)
	r := m["result"].(map[string]any)
	if r["png"] == nil || r["width"] != 3.0 || r["height"] != 2.0 {
		t.Errorf("capture: %v", r)
	}
	var cr captureResult
	b, _ := json.Marshal(m["result"])
	if json.Unmarshal(b, &cr) != nil || !bytes.Equal(cr.PNG, buf.Bytes()) {
		t.Errorf("the PNG did not come back whole")
	}
	if m := do(t, d, `{"id": 1, "method": "eval", "params": {"script": "({title: document.title})"}}`); m["result"].(map[string]any)["value"].(map[string]any)["title"] != "E2E" {
		t.Errorf("eval: %v", m)
	}
	if m := do(t, d, `{"id": 1, "method": "eval", "params": {"script": "throw 1"}}`); errCode(m) != codeFailed {
		t.Errorf("eval that throws: %v", m)
	}
	if m := do(t, d, `{"id": 1, "method": "idle", "params": {"timeout": 1500}}`); m["error"] != nil || d.waited != 1500*time.Millisecond {
		t.Errorf("idle: %v, waited %v", m, d.waited)
	}
}

func TestNames(t *testing.T) {
	if len(roleNames) != int(platform.RoleInsertion)+1 {
		t.Errorf("%d role names for %d roles", len(roleNames), platform.RoleInsertion+1)
	}
	if platform.AccessStates(1)<<(len(stateNames)-1) != platform.AccessHorizontal {
		t.Errorf("%d state names", len(stateNames))
	}
	if platform.AccessActions(1)<<(len(actionNames)-1) != platform.ActionExpand {
		t.Errorf("%d action names", len(actionNames))
	}
	seen := map[string]bool{}
	for _, n := range roleNames {
		if seen[n] {
			t.Errorf("role name %q twice", n)
		}
		seen[n] = true
	}
	if len(actionKinds) != int(platform.AccessInsertText)+1 {
		t.Errorf("%d actions for %d", len(actionKinds), platform.AccessInsertText+1)
	}
}

// socketPath returns a path for a socket, short enough for macOS's limit
// of 104 bytes.
func socketPath(t *testing.T) string {
	dir, err := os.MkdirTemp("", "mygo-automation")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "s")
}

func TestServe(t *testing.T) {
	path := socketPath(t)
	l, err := listen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("socket's mode %v, %v", fi.Mode(), err)
		}
	}
	tree, keys := testTree()
	go serve(l, &fakeDriver{access: tree, keys: keys})

	c, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	// Requests answered in order, each on a line of its own.
	if _, err := c.Write([]byte("{\"id\": 1, \"method\": \"tree\"}\n\n{\"id\": 2, \"method\": \"nope\"}\n")); err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(c)
	for i, want := range []string{`"id":1,"result":{"window":1,`, `"id":2,"error":{"code":"unknown_method"`} {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(line, want) {
			t.Errorf("response %d: %s", i+1, line)
		}
	}

	// Another app's socket is left alone, as is a file that is no socket.
	if l2, err := listen(path); err == nil {
		l2.Close()
		t.Error("listened on a socket another app listens on")
	}
	file := filepath.Join(filepath.Dir(path), "f")
	os.WriteFile(file, nil, 0o600)
	if l2, err := listen(file); err == nil {
		l2.Close()
		t.Error("listened in place of a file")
	}
}

// TestStaleSocket listens where an app that ended left its socket.
func TestStaleSocket(t *testing.T) {
	path := socketPath(t)
	l, err := listen(path)
	if err != nil {
		t.Fatal(err)
	}
	l.(*net.UnixListener).SetUnlinkOnClose(false)
	l.Close()
	if _, err := os.Lstat(path); err != nil {
		t.Fatal("no socket left: ", err)
	}
	l, err = listen(path)
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("the socket stays once closed: %v", err)
	}
}
