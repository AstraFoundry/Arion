package application

import (
	"arion.local/core/domain"
	"encoding/json"
	"testing"
)

func TestClipboardFilesOptInAndLegacyPreferences(t *testing.T) {
	p := DefaultClipboardPreferences()
	if err := json.Unmarshal([]byte(`{"enabled":true}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.FilesEnabled {
		t.Fatal("missing file preference inferred file opt-in")
	}
	items := []domain.Clip{
		{Payload: domain.ClipPayload{Kind: domain.ClipFiles}, Pinned: true},
		{Payload: domain.ClipPayload{Kind: domain.ClipText}},
		{Payload: domain.ClipPayload{Kind: domain.ClipImage}},
		{Payload: domain.ClipPayload{Kind: domain.ClipLink}},
	}
	visible := p.VisibleItems(items)
	if len(visible) != 3 || items[0].Payload.Kind != domain.ClipFiles {
		t.Fatal("filter mutated storage or removed other kinds")
	}
	p.FilesEnabled = true
	if len(p.VisibleItems(items)) != 4 {
		t.Fatal("re-enabled files not restored")
	}
}
