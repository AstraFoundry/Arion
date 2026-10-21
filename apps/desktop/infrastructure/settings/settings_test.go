package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
	if err != nil || len(repairs) != 0 || !reflect.DeepEqual(loaded, prefs) {
		t.Fatalf("round trip: %v %v %v", loaded, repairs, err)
	}
	if err := os.WriteFile(store.Path, []byte(`{"version":1,"sounds":false,"shortcut":"Cmd+Shift+D"}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, repairs, err = store.Load()
	if err != nil || len(repairs) != 0 || loaded.UILanguage != application.English || loaded.Sounds || loaded.HoldKey != application.DefaultHoldKey(runtime.GOOS) || !loaded.CheckUpdates {
		t.Fatalf("legacy profile: %v %v %v", loaded, repairs, err)
	}
	// The retired toggle shortcut left profiles with the hold key off; they
	// move to this platform's default key.
	if err := os.WriteFile(store.Path, []byte(`{"version":1,"shortcut":"Alt+Space","holdKey":""}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, repairs, err = store.Load()
	if err != nil || loaded.HoldKey != application.DefaultHoldKey(runtime.GOOS) || !slices.Contains(repairs, "holdKey") {
		t.Fatalf("hold key off: %v %v %v", loaded, repairs, err)
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

func TestLegacyInputModeIsRetired(t *testing.T) {
	for _, mode := range []string{"shortcut", "system_ime"} {
		store := Store{Path: filepath.Join(t.TempDir(), "preferences.json")}
		if err := os.WriteFile(store.Path, []byte(`{"version":1,"mode":"`+mode+`","shortcut":"Cmd+Shift+D","sounds":false}`), 0600); err != nil {
			t.Fatal(err)
		}
		prefs, repairs, err := store.Load()
		if err != nil || len(repairs) != 0 || prefs.Sounds {
			t.Fatalf("legacy migration: %+v %v %v", prefs, repairs, err)
		}
		if err := store.Save(prefs); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(store.Path)
		if err != nil {
			t.Fatal(err)
		}
		var saved map[string]any
		if err := json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		for _, retired := range []string{"mode", "shortcut"} {
			if _, exists := saved[retired]; exists {
				t.Fatalf("retired %s persisted again", retired)
			}
		}
	}
}

func TestPrereleaseUpdatesPersistence(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "preferences.json")}
	prefs, _, err := store.Load()
	if err != nil || prefs.AllowPrereleaseUpdates {
		t.Fatalf("default: %+v %v", prefs, err)
	}
	prefs.AllowPrereleaseUpdates = true
	if err := store.Save(prefs); err != nil {
		t.Fatal(err)
	}
	got, _, err := store.Load()
	if err != nil || !got.AllowPrereleaseUpdates {
		t.Fatalf("round trip: %+v %v", got, err)
	}
}
