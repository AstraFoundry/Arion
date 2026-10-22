package settings

import (
	"testing"

	"arion.local/desktop/ui/shared/copy"
)

// TestSectionsTable keeps the one list of Settings sections well formed:
// unique IDs, a translated title in each language, an icon for every
// sidebar item and the account as the only card.
func TestSectionsTable(t *testing.T) {
	seen := map[string]bool{}
	cards := 0
	for _, s := range sections {
		if s.id == "" || seen[s.id] {
			t.Fatalf("section ID %q empty or repeated", s.id)
		}
		seen[s.id] = true
		for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
			if copy.For(locale).Text(s.title) == "" {
				t.Errorf("%s: no %s title", s.id, locale)
			}
		}
		if s.card {
			cards++
			if s.id != Account {
				t.Errorf("%s drawn as the account card", s.id)
			}
		} else if s.icon == nil {
			t.Errorf("%s: sidebar item without an icon", s.id)
		}
	}
	if cards != 1 {
		t.Errorf("%d account cards", cards)
	}
	for _, id := range []string{Overview, General, Dictation, Keyboard, Microphone, Model, Vocabulary, Account, About} {
		if !seen[id] {
			t.Errorf("section %s missing from the table", id)
		}
	}
}
