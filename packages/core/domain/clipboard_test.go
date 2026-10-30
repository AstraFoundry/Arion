package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClipboardDedupPinsAndExactText(t *testing.T) {
	now := time.Unix(10000000, 0)
	l := DefaultClipLimits()
	p := ClipTextPayload(" 你好\nArion\r\n🙂 ")
	items, err := CaptureClip(nil, p, l, now)
	if err != nil {
		t.Fatal(err)
	}
	id := items[0].ID
	items[0].Pinned = true
	items, err = CaptureClip(items, ClipTextPayload("other"), l, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	items, err = CaptureClip(items, p, l, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != id || !items[0].Pinned || items[0].Payload.Text != p.Text || !items[0].Copied.Equal(now.Add(2*time.Second)) {
		t.Fatalf("dedup lost identity/payload: %+v", items)
	}
	if p.Kind != ClipText {
		t.Fatal("wrong text classification")
	}
}

func TestClipboardRetentionProtectsPinsAndRejectsCapacity(t *testing.T) {
	now := time.Unix(10000000, 0)
	l := DefaultClipLimits()
	l.Count = 1
	l.Days = 1
	l.Bytes = 1 << 20
	l.ItemBytes = 1 << 20
	pinned := Clip{ID: "pin", Payload: ClipTextPayload(strings.Repeat("p", 700000)), Copied: now.Add(-48 * time.Hour), Pinned: true}
	old := Clip{ID: "old", Payload: ClipTextPayload("old"), Copied: now.Add(-48 * time.Hour)}
	a := Clip{ID: "a", Payload: ClipTextPayload("a"), Copied: now.Add(-time.Hour)}
	b := Clip{ID: "b", Payload: ClipTextPayload("b"), Copied: now}
	got, err := PruneClips([]Clip{a, pinned, old, b}, l, now)
	if err != nil || len(got) != 2 || got[0].ID != "pin" || got[1].ID != "b" {
		t.Fatalf("retention: %v %v", got, err)
	}
	_, err = CaptureClip(got, ClipTextPayload(strings.Repeat("x", 700000)), l, now.Add(time.Second))
	if !errors.Is(err, ErrClipCapacity) {
		t.Fatalf("capacity: %v", err)
	}
	_, err = CaptureClip(got, ClipTextPayload(strings.Repeat("x", int(l.ItemBytes)+1)), l, now)
	if !errors.Is(err, ErrClipOversize) {
		t.Fatalf("oversize: %v", err)
	}
	pinned.Payload = ClipTextPayload(strings.Repeat("x", int(l.Bytes)+1))
	if _, err = PruneClips([]Clip{pinned}, l, now); !errors.Is(err, ErrClipCapacity) {
		t.Fatalf("pin evicted: %v", err)
	}
}

func TestClipboardFiltersAndFileOrdering(t *testing.T) {
	now := time.Now()
	l := DefaultClipLimits()
	paths := []string{"/sample/项目/Plan.md", "/sample/image.png"}
	files := ClipPayload{Kind: ClipFiles, Files: paths}
	items, err := CaptureClip(nil, files, l, now)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(items[0].Payload.Files, paths) {
		t.Fatal("file group lost")
	}
	if len(FindClips(items, "PLAN", ClipFiles)) != 1 || len(FindClips(items, "", ClipText)) != 0 {
		t.Fatal("filter mismatch")
	}
	reversed := ClipPayload{Kind: ClipFiles, Files: []string{paths[1], paths[0]}}
	if reversed.ID() == files.ID() {
		t.Fatal("ordered file groups must differ")
	}
	if _, err := CaptureClip(nil, ClipPayload{Kind: ClipText, Text: "x\x00y"}, l, now); err == nil {
		t.Fatal("NUL text accepted")
	}
}
