package usage

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"arion.local/core/domain"
)

func TestUsageRoundTrip(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "usage.json")}
	days, err := store.Load()
	if err != nil || days != nil {
		t.Fatalf("first run: %v %v", days, err)
	}
	days = days.Add(time.Date(2026, 10, 14, 9, 0, 0, 0, time.Local), domain.Usage{SpeechMs: 4000, Words: 9, Scene: domain.SceneWork})
	if err := store.Save(days); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil || !reflect.DeepEqual(loaded, days) {
		t.Fatalf("round trip: %+v %v", loaded, err)
	}
	if err := os.WriteFile(store.Path, []byte(`{"version":9,"days":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("unknown version accepted")
	}
}
