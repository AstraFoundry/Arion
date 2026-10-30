package vocabulary

import (
	"testing"

	"arion.local/desktop/ui/shared/runtime"
)

func TestGroupsSearchAndIdentity(t *testing.T) {
	entries := []runtime.Replacement{
		{ID: "one", From: "cloud code", To: "Claude Code"},
		{ID: "two", From: "a rion", To: "Arion"},
		{ID: "three", From: "clawed code", To: "Claude Code"},
	}
	for _, query := range []string{" CLOUD ", "CLAUDE"} {
		groups := Groups(entries, query)
		if len(groups) != 1 || groups[0].Text != "Claude Code" || len(groups[0].Variants) != 2 || groups[0].Variants[1].ID != "three" {
			t.Fatalf("query %q: %#v", query, groups)
		}
	}
	groups := Groups(entries, "")
	if len(groups) != 2 || groups[0].Text != "Arion" || groups[1].Text != "Claude Code" || len(Groups(entries, "unmatched")) != 0 {
		t.Fatalf("newest term should lead and unmatched search should be empty: %#v", groups)
	}
}

func TestCorrectOnlyRecordJoinsItsTerm(t *testing.T) {
	groups := Groups([]runtime.Replacement{
		{ID: "pair", From: "ary on", To: "Arion"},
		{ID: "word", To: "Arion", Hotword: true},
	}, "Arion")
	if len(groups) != 1 || len(groups[0].Records) != 2 || len(groups[0].Variants) != 1 || groups[0].Variants[0].ID != "pair" {
		t.Fatalf("group %#v", groups)
	}
	if len(groups[0].Readings) != 1 || groups[0].Readings[0].ID != "word" {
		t.Fatalf("a new reading should bind to the correct-only record: %#v", groups[0].Readings)
	}
}

func TestReadingsKeepEveryBoundRecord(t *testing.T) {
	groups := Groups([]runtime.Replacement{
		{ID: "first", From: "星茶", To: "星槎"},
		{ID: "second", From: "星叉", To: "星槎"},
	}, "")
	if len(groups[0].Readings) != 1 || groups[0].Readings[0].ID != "first" {
		t.Fatalf("a term without a correct-only record binds to its first variant: %#v", groups[0].Readings)
	}
	groups = Groups([]runtime.Replacement{
		{ID: "word", To: "星槎", KeyboardPinyin: "xing cha"},
		{ID: "pair", From: "星茶", To: "星槎", KeyboardPinyin: "xing zha"},
	}, "")
	if readings := groups[0].Readings; len(readings) != 2 || readings[0].ID != "word" || readings[1].ID != "pair" {
		t.Fatalf("every bound reading should stay visible: %#v", readings)
	}
}
