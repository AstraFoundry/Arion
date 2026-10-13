package settings

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"arion.local/core/application"
)

func TestLanguagePersistenceAndLegacyProfiles(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "preferences.json")}
	prefs, repairs, err := store.Load()
	if err != nil || len(repairs) != 0 || prefs.UILanguage != application.English {
		t.Fatalf("first run: %v %v %v", prefs, repairs, err)
	}
	prefs.UILanguage = application.SimplifiedChinese
	if err := store.Save(prefs); err != nil {
		t.Fatal(err)
	}
	loaded, repairs, err := store.Load()
	if err != nil || len(repairs) != 0 || loaded != prefs {
		t.Fatalf("round trip: %v %v %v", loaded, repairs, err)
	}
	if err := os.WriteFile(store.Path, []byte(`{"version":1,"sounds":false,"shortcut":"Cmd+Shift+D"}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, repairs, err = store.Load()
	if err != nil || len(repairs) != 0 || loaded.UILanguage != application.English || loaded.Sounds || loaded.Shortcut != "Cmd+Shift+D" {
		t.Fatalf("legacy profile: %v %v %v", loaded, repairs, err)
	}
	if err := os.WriteFile(store.Path, []byte(`{"version":1,"uiLanguage":"fr"}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, repairs, err = store.Load()
	if err != nil || loaded.UILanguage != application.English || !slices.Contains(repairs, "uiLanguage") {
		t.Fatalf("invalid locale: %v %v %v", loaded, repairs, err)
	}
	if err := os.WriteFile(store.Path, []byte(`{"uiLanguage":42}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Load(); err == nil {
		t.Fatal("malformed preference accepted")
	}
}
