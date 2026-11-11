package automationcli

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func nodes(spec ...string) []node {
	var out []node
	for _, s := range spec {
		id, label, _ := strings.Cut(s, ":")
		out = append(out, node{ID: id, Role: "text", Label: label})
	}
	return out
}

func diffText(lines []diffLine) []string {
	var out []string
	for _, l := range lines {
		out = append(out, l.String())
	}
	return out
}

// TestRefs checks the life of refs: elements keep theirs while windows
// show them, new ones get new numbers, and the refs of elements gone are
// rejected, never given to another element.
func TestRefs(t *testing.T) {
	s := &session{}
	s.observe(&tree{Window: 1, Nodes: nodes("10:a", "11:b", "12:c")})
	s.observe(&tree{Window: 2, Nodes: nodes("10:other window")})
	want := func(n, window int, id string) {
		t.Helper()
		w, got, err := s.lookup(n)
		if err != nil || w != window || got != id {
			t.Errorf("%s: window %d, element %s, %v; want %d, %s", ref(n), w, got, err, window, id)
		}
	}
	stale := func(n int, why string) {
		t.Helper()
		_, _, err := s.lookup(n)
		var se *staleError
		if !errors.As(err, &se) || !strings.Contains(err.Error(), why) {
			t.Errorf("%s: %v, want stale: %s", ref(n), err, why)
		}
	}
	want(1, 1, "10")
	want(2, 1, "11")
	want(3, 1, "12")
	want(4, 2, "10")
	stale(5, "no snapshot or diff")

	// 11 goes, 13 comes; 12 moves before 10.
	before := s.observe(&tree{Window: 1, Nodes: nodes("12:c", "10:a", "13:d")})
	if before == nil || len(before.Nodes) != 3 {
		t.Fatalf("observe returned %+v", before)
	}
	want(1, 1, "10")
	want(3, 1, "12")
	want(5, 1, "13")
	stale(2, "gone")
	want(4, 2, "10")

	// 11 coming back is a new element to the session: the old ref stays
	// stale, so that a ref read before never acts on what is there now.
	s.observe(&tree{Window: 1, Nodes: nodes("11:b")})
	want(6, 1, "11")
	stale(2, "gone")
	stale(1, "gone")

	s.forget(2)
	stale(4, "gone")
	if s.Next != 7 {
		t.Errorf("next ref %d", s.Next)
	}
}

func TestParseRef(t *testing.T) {
	for s, want := range map[string]int{"@e1": 1, "e12": 12, "@e0": 0, "@e": 0, "@x1": 0, "@e01": 0, "@e-1": 0, "e1a": 0, "key=a": 0} {
		if n, ok := parseRef(s); n != want || ok != (want > 0) {
			t.Errorf("parseRef(%q) = %d, %v", s, n, ok)
		}
	}
}

func TestSessionFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := sessionPath(dir, "../x"); err == nil {
		t.Error("a session name with a path")
	}
	if _, err := loadSession(dir, "a"); err == nil || !strings.Contains(err.Error(), "open") {
		t.Errorf("loading no session: %v", err)
	}
	path, _ := sessionPath(dir, "a")
	s := &session{Socket: "/tmp/app.sock", Window: 2, Next: 1, name: "a", path: path}
	s.observe(&tree{Window: 2, Focus: "10", Nodes: nodes("10:a")})
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	// Another session keeps refs of its own.
	other := &session{Socket: "/tmp/app.sock", Next: 1, name: "b"}
	other.path, _ = sessionPath(dir, "b")
	other.observe(&tree{Window: 2, Nodes: nodes("20:x", "10:a")})
	if err := other.save(); err != nil {
		t.Fatal(err)
	}
	got, err := loadSession(dir, "a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Socket != s.Socket || got.Window != 2 || got.Next != 2 || got.Seen["2"].Focus != "10" {
		t.Errorf("loaded %+v", got)
	}
	if w, id, err := got.lookup(1); w != 2 || id != "10" || err != nil {
		t.Errorf("@e1 after loading: %d %s %v", w, id, err)
	}
}

func TestDiff(t *testing.T) {
	s := &session{}
	s.observe(&tree{Window: 1, Nodes: nodes("1:title", "2:a", "3:b", "4:c", "5:d", "6:e", "7:f", "8:g")})
	// b changes, e goes, x comes after f; focus moves to g.
	before := s.observe(&tree{Window: 1, Focus: "8", Nodes: nodes("1:title", "2:a", "3:B", "4:c", "5:d", "7:f", "9:x", "8:g")})
	got := diffText(diff(before, s.Seen["1"], 1))
	want := []string{
		`= @e2 [text] "a"`,
		`- @e3 [text] "b"`,
		`+ @e3 [text] "B"`,
		`= @e4 [text] "c"`,
		`= @e5 [text] "d"`,
		`- @e6 [text] "e"`,
		`= @e7 [text] "f"`,
		`+ @e9 [text] "x"`,
		`- @e8 [text] "g"`,
		`+ @e8 [text] "g" focused`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("diff:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// Nothing changed.
	before = s.observe(&tree{Window: 1, Focus: "8", Nodes: nodes("1:title", "2:a", "3:B", "4:c", "5:d", "7:f", "9:x", "8:g")})
	if got := diff(before, s.Seen["1"], 1); len(got) != 0 {
		t.Errorf("diff of the same: %v", diffText(got))
	}

	// Moves alone change nothing; a window seen for the first time is all
	// new.
	before = s.observe(&tree{Window: 1, Focus: "8", Nodes: nodes("8:g", "1:title", "2:a", "3:B", "4:c", "5:d", "7:f", "9:x")})
	if got := diff(before, s.Seen["1"], 1); len(got) != 0 {
		t.Errorf("diff of a move: %v", diffText(got))
	}
	before = s.observe(&tree{Window: 3, Nodes: nodes("1:new")})
	if got := diffText(diff(before, s.Seen["3"], 1)); !slices.Equal(got, []string{`+ @e10 [text] "new"`}) {
		t.Errorf("diff of a new window: %v", got)
	}

	// Everything gone.
	before = s.observe(&tree{Window: 3})
	if got := diffText(diff(before, s.Seen["3"], 1)); !slices.Equal(got, []string{`- @e10 [text] "new"`}) {
		t.Errorf("diff of an emptied window: %v", got)
	}
}
