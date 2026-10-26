package application

import (
	"encoding/json"
	"testing"
)

func TestProductNotificationsAreExplicitAndPersisted(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		p := DefaultPreferences(platform)
		if p.ProductNotifications {
			t.Fatal("new profile opted in")
		}
		if json.Unmarshal([]byte(`{"version":1}`), &p) != nil || p.ProductNotifications {
			t.Fatal("migration opted in")
		}
		p.ProductNotifications = true
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var loaded Preferences
		if json.Unmarshal(data, &loaded) != nil || !loaded.ProductNotifications {
			t.Fatal("preference did not persist")
		}
	}
}
