package updatestate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStatePersistsStagingAndReminderDecisions(t *testing.T) {
	s := Store{Path: filepath.Join(t.TempDir(), "updates.json")}
	st, err := s.Load()
	if err != nil || st.Schema != 1 {
		t.Fatalf("new state: %+v %v", st, err)
	}
	st.Stage = "/owned/stage/record.json"
	st.Version = "1.2.3"
	st.Beta = true
	st.Automatic = true
	st.Reminders = []Reminder{{Key: "stable.1.2.3.ready", Submitted: time.Now().UTC(), Later: time.Now().UTC().Add(time.Hour)}}
	if err = s.Save(st); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil || got.Stage != st.Stage || got.Version != st.Version || got.Reminders[0].Later != st.Reminders[0].Later {
		t.Fatalf("round trip: %+v %v", got, err)
	}
}
func TestStateRejectsCorruptionRatherThanLosingSuppression(t *testing.T) {
	s := Store{Path: filepath.Join(t.TempDir(), "updates.json")}
	for _, data := range []string{`{`, `{"schema":2}`, `{"schema":1,"stage":"/tmp/stage"}`, `{"schema":1,"unknown":true}`} {
		if err := os.WriteFile(s.Path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Load(); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}
