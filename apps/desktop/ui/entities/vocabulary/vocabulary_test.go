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
	if len(groups) != 2 || groups[0].Text != "Claude Code" || groups[1].Text != "Arion" || len(Groups(entries, "unmatched")) != 0 {
		t.Fatalf("unexpected group ordering or empty search: %#v", groups)
	}
}

func TestHotwordsGroupWithoutEmptyAlias(t *testing.T) {
	groups := Groups([]runtime.Replacement{
		{ID: "word", To: "Arion", Hotword: true},
		{ID: "pair", From: "ary on", To: "Arion", Hotword: true},
	}, "Arion")
	if len(groups) != 1 || !groups[0].Hotword || len(groups[0].Variants) != 1 || len(groups[0].HotwordIDs) != 1 || groups[0].HotwordIDs[0] != "word" {
		t.Fatalf("group %#v", groups)
	}
}
