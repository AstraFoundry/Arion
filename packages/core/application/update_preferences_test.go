package application

import (
	"encoding/json"
	"testing"
)

func TestUpdatePreferencesPreserveExistingConsent(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		p := DefaultPreferences(platform)
		if !p.CheckUpdates || p.AutoInstallUpdates {
			t.Fatalf("%s: new profile update defaults: %+v", platform, p)
		}
		if err := json.Unmarshal([]byte(`{"version":1,"checkUpdates":false,"allowPrereleaseUpdates":true}`), &p); err != nil {
			t.Fatal(err)
		}
		p, _ = p.Normalize(platform)
		if p.CheckUpdates || p.AutoInstallUpdates || !p.AllowPrereleaseUpdates {
			t.Fatalf("%s: migrated profile gained consent or lost channel: %+v", platform, p)
		}
		p.AutoInstallUpdates = true
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		loaded := DefaultPreferences(platform)
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatal(err)
		}
		loaded, repaired := loaded.Normalize(platform)
		if loaded.CheckUpdates || !loaded.AutoInstallUpdates || len(repaired) != 0 {
			t.Fatalf("%s: independent choices failed round trip: %+v repairs=%v", platform, loaded, repaired)
		}
	}
}
