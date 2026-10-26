package application

import (
	"encoding/json"
	"testing"
)

func TestLegacyNotificationSwitchesAreIgnored(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		for _, legacy := range []string{
			`{"version":1,"productNotifications":false,"systemUpdateReminders":false}`,
			`{"version":1,"productNotifications":true,"systemUpdateReminders":true}`,
		} {
			p := DefaultPreferences(platform)
			if err := json.Unmarshal([]byte(legacy), &p); err != nil {
				t.Fatal(err)
			}
			p, _ = p.Normalize(platform)
			encoded, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"productNotifications", "systemUpdateReminders"} {
				if _, exists := fields[key]; exists {
					t.Fatalf("%s retained obsolete notification preference %s", platform, key)
				}
			}
		}
	}
}
