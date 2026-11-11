package automationcli

import (
	"encoding/json"
	"strings"
	"testing"
)

func f(v float64) *float64 { return &v }

func TestLine(t *testing.T) {
	for _, c := range []struct {
		n    node
		full bool
		want string
	}{
		{node{ID: "1", Role: "text", Label: "Count: 0"}, false, `[text] "Count: 0"`},
		{node{ID: "2", Role: "button", Label: "Add", Key: json.RawMessage(`"add"`), States: []string{"focusable", "disabled"}, Actions: []string{"press"}}, false, `[button] "Add" key=add disabled`},
		// The focused element, a value, and a placeholder only under no label.
		{node{ID: "3", Role: "textField", Label: "Name", Value: "Ada", Placeholder: "Your name"}, false, `[textField] "Name" value="Ada" focused`},
		{node{ID: "4", Role: "textField", Placeholder: "Search"}, false, `[textField] placeholder="Search"`},
		// Keys that read as other values, quoted; numbers as themselves.
		{node{ID: "5", Role: "button", Key: json.RawMessage(`"true"`)}, false, `[button] key="true"`},
		{node{ID: "5", Role: "button", Key: json.RawMessage(`"a b"`)}, false, `[button] key="a b"`},
		{node{ID: "5", Role: "listItem", Key: json.RawMessage(`7`)}, false, `[listItem] key=7`},
		// A slider's value, when it has no text for it.
		{node{ID: "6", Role: "slider", Label: "Volume", Min: f(0), Max: f(1), Now: f(0.25), Step: f(0.05)}, false, `[slider] "Volume" value=0.25`},
		{node{ID: "6", Role: "slider", Label: "Volume", Min: f(0), Max: f(1), Now: f(0.25), Step: f(0.05)}, true, `[slider] "Volume" value=0.25 range=0..1 step=0.05 bounds=0,0,0x0 id=6`},
		{node{ID: "7", Role: "group", WebView: true}, false, `[group] webview`},
		// What its suggestions complete, after its states.
		{node{ID: "10", Role: "comboBox", Label: "City", States: []string{"focusable", "expanded"}, Complete: "list"}, false, `[comboBox] "City" expanded autocomplete=list`},
		// The keys that press it, after its key.
		{node{ID: "9", Role: "button", Label: "Redo", Key: json.RawMessage(`"redo"`), Shortcuts: "Ctrl+Shift+Z Ctrl+Y"}, false, `[button] "Redo" key=redo keyshortcuts="Ctrl+Shift+Z Ctrl+Y"`},
		{
			node{ID: "18446744073709551615", Role: "textField", Label: "Name", Value: "Ada", Placeholder: "Your name", Description: "Who", Selection: &[2]int{1, 3}, Actions: []string{"focus", "setValue"}, Bounds: rect{20, 90.5, 360, 32}},
			true,
			`[textField] "Name" value="Ada" placeholder="Your name" description="Who" selection=1,3 actions=focus,setValue bounds=20,90.5,360x32 id=18446744073709551615`,
		},
		{node{ID: "8", Role: "treeItem", Label: "src", Level: 2, PosInSet: 1, SetSize: 3, States: []string{"expandable", "expanded"}}, true, `[treeItem] "src" expandable expanded level=2 item=1/3 bounds=0,0,0x0 id=8`},
		// A grid's size and a row's place in it, as the app gave them.
		{node{ID: "10", Role: "grid", Label: "Sheet", RowCount: 10001, ColCount: 5, States: []string{"multiselectable"}}, true, `[grid] "Sheet" multiselectable rowcount=10001 colcount=5 bounds=0,0,0x0 id=10`},
		{node{ID: "11", Role: "row", RowIndex: 4122}, true, `[row] rowindex=4122 bounds=0,0,0x0 id=11`},
		{node{ID: "11", Role: "row", RowIndex: 4122}, false, `[row]`},
		{node{ID: "12", Role: "cell", Label: "Total", ColSpan: 2}, true, `[cell] "Total" colspan=2 bounds=0,0,0x0 id=12`},
	} {
		if got := line(&c.n, "3", c.full); got != c.want {
			t.Errorf("line(%+v, full %v)\n got %s\nwant %s", c.n, c.full, got, c.want)
		}
	}
}

func TestLongText(t *testing.T) {
	long := strings.Repeat("é", maxText+30)
	n := node{ID: "1", Role: "textField", Value: long}
	want := `[textField] value="` + strings.Repeat("é", maxText) + `…" (30 more)`
	if got := line(&n, "", false); got != want {
		t.Errorf("got %s", got)
	}
	if got := line(&n, "", true); !strings.Contains(got, `value="`+long+`"`) {
		t.Errorf("full cuts the text: %s", got)
	}
	// Lines stay lines.
	n.Value = "a\nb"
	if got := line(&n, "", false); got != `[textField] value="a\nb"` {
		t.Errorf("got %s", got)
	}
}

// TestKeys checks that a key, as lines write it, is a selector that
// names it.
func TestKeys(t *testing.T) {
	for _, k := range []string{`"add"`, `"a b"`, `"true"`, `"5"`, `5`, `true`, `"row/1"`, `"null"`, `""`} {
		text := keyText(json.RawMessage(k))
		if got := string(keyJSON(text)); got != k {
			t.Errorf("key %s writes %s, which selects %s", k, text, got)
		}
	}
}

func TestSnapshot(t *testing.T) {
	sn := &seen{Focus: "4", Refs: map[string]int{"1": 1, "2": 2, "3": 3, "4": 4, "5": 5}, Nodes: []node{
		{ID: "1", Role: "dialog", Label: "Settings"},
		{ID: "2", Parent: "1", Role: "text", Label: "Theme"},
		{ID: "3", Parent: "1", Role: "group"},
		{ID: "4", Parent: "3", Role: "checkBox", Label: "Dark", States: []string{"focusable", "checked"}, Actions: []string{"press", "focus"}},
		{ID: "5", Role: "button", Label: "Close", Actions: []string{"press"}},
	}}
	var b strings.Builder
	writeSnapshot(&b, sn, []string{"Saved"}, false, false)
	want := `@e1 [dialog] "Settings"
  @e2 [text] "Theme"
  @e3 [group]
    @e4 [checkBox] "Dark" checked focused
@e5 [button] "Close"
announced "Saved"
`
	if b.String() != want {
		t.Errorf("snapshot:\n%s\nwant:\n%s", b.String(), want)
	}
	b.Reset()
	writeSnapshot(&b, sn, nil, true, false)
	want = `@e4 [checkBox] "Dark" checked focused
@e5 [button] "Close"
`
	if b.String() != want {
		t.Errorf("snapshot -i:\n%s\nwant:\n%s", b.String(), want)
	}
	b.Reset()
	writeSnapshot(&b, &seen{Nodes: sn.Nodes[:3], Refs: sn.Refs}, nil, true, false)
	if !strings.HasPrefix(b.String(), "(no interactive elements") {
		t.Errorf("snapshot -i of no controls: %s", b.String())
	}
}

func TestWindowLine(t *testing.T) {
	if got := windowLine(windowInfo{ID: 2, Title: "Docs", Kind: "page", URL: "http://localhost:5173/", Visible: false}, true); got != `w2 [page] "Docs" url=http://localhost:5173/ hidden selected` {
		t.Errorf("got %s", got)
	}
	if got := windowLine(windowInfo{ID: 1, Title: "Counter", Kind: "content", Focused: true, Visible: true}, false); got != `w1 [content] "Counter" focused` {
		t.Errorf("got %s", got)
	}
}
