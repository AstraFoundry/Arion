package editor

import (
	"math/rand/v2"
	"strings"
	"testing"
)

func TestCodeBufferEditUndo(t *testing.T) {
	b := NewBuffer("hello\nworld\n")
	if b.LineCount() != 3 || b.Line(1) != "world" || b.Modified() {
		t.Fatalf("lines %d, %q, modified %v", b.LineCount(), b.Line(1), b.Modified())
	}
	b.Edit(Edit{Start: 0, End: 5, Text: "goodbye"}, Edit{Start: 6, End: 6, Text: "big "})
	if got := b.String(); got != "goodbye\nbig world\n" {
		t.Fatalf("String = %q", got)
	}
	if !b.Modified() || !b.CanUndo() {
		t.Fatal("not modified after an edit")
	}
	b.Undo()
	if got := b.String(); got != "hello\nworld\n" || b.Modified() {
		t.Fatalf("after undo %q, modified %v", got, b.Modified())
	}
	b.Redo()
	if got := b.String(); got != "goodbye\nbig world\n" {
		t.Fatalf("after redo %q", got)
	}
	if p := b.Pos(12); p != (Pos{1, 4}) {
		t.Fatalf("Pos = %+v", p)
	}
	if o := b.Offset(Pos{1, 100}); o != 17 {
		t.Fatalf("Offset = %d", o)
	}
	b.Transact(func() {
		b.Edit(Edit{Start: 0, End: 0, Text: "a"})
		b.Edit(Edit{Start: 0, End: 0, Text: "b"})
	})
	b.Undo()
	if got := b.String(); got != "goodbye\nbig world\n" {
		t.Fatalf("Transact undo %q", got)
	}
}

// The log moves offsets and lines through edits as the text moved.
func TestCodeBufferLog(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	b := NewBuffer("one\ntwo\nthree\nfour\nfive\n")
	for range 500 {
		before, v := b.snapshot()
		text := before.String()
		// A marker line, to find where line moved.
		at := r.IntN(len(text) + 1)
		end := min(len(text), at+r.IntN(6))
		ins := []string{"", "x", "\n", "ab\ncd", "\n\n"}[r.IntN(5)]
		probe := r.IntN(len(text) + 1)
		b.Edit(Edit{Start: at, End: end, Text: ins})
		if b.String() == text {
			continue
		}
		changes, _, ok := b.changesSince(v)
		if !ok || len(changes) != 1 {
			t.Fatalf("changesSince = %v, %v", changes, ok)
		}
		got := mapOffset(probe, changes[0], biasRight)
		var want int
		switch {
		case probe < at:
			want = probe
		case probe >= end && (probe > at || end > at):
			want = probe + len(ins) - (end - at)
		default:
			want = at + len(ins)
		}
		if probe > at && probe < end {
			want = at + len(ins)
		}
		if got != want {
			t.Fatalf("mapOffset(%d) through [%d,%d)->%q = %d, want %d", probe, at, end, ins, got, want)
		}
		after, _ := b.snapshot()
		// Lines wholly after the change keep their text.
		pl := before.LineOf(probe)
		if before.LineStart(pl) > end && pl > before.LineOf(end) {
			if nl := mapLine(pl, changes[0]); after.Line(nl) != before.Line(pl) {
				t.Fatalf("mapLine(%d) = %d: %q, want %q", pl, nl, after.Line(nl), before.Line(pl))
			}
		}
		if after.LineCount() != strings.Count(after.String(), "\n")+1 {
			t.Fatal("line count")
		}
	}
}
