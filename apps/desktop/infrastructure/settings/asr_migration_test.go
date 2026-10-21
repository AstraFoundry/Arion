package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRetiredASROverrideMigration(t *testing.T) {
	for _, id := range []string{"xasr-160-xasr-final", "xasr-160-qwen3asr-17-final", "qwen3-asr-1.7b-mlx-8bit", "unknown-model"} {
		path := filepath.Join(t.TempDir(), "preferences.json")
		if err := os.WriteFile(path, []byte(`{"modelOverride":"`+id+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
		prefs, _, err := (Store{Path: path}).Load()
		if err != nil {
			t.Fatal(err)
		}
		retired := id == "xasr-160-xasr-final" || id == "xasr-160-qwen3asr-17-final"
		if retired && prefs.ModelOverride != "" {
			t.Fatalf("retired override retained: %s", id)
		}
		if !retired && prefs.ModelOverride != id {
			t.Fatalf("unrelated override repaired: %s", id)
		}
	}
}
