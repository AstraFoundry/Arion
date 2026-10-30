package usageupload

import (
	"arion.local/core/domain"
	"testing"
	"time"
)

func TestLedgerCountsCommittedUsageAndRetainsNewerUnacknowledgedCounts(t *testing.T) {
	ledger := Ledger{Days: map[string]Day{}, Acked: map[string]Day{}}
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.FixedZone("local", 8*3600))
	if err := ledger.Record(now, nil); err != nil {
		t.Fatal(err)
	}
	usage := domain.Usage{Words: 120, SpeechMs: 60000, Scene: "test-scene"}
	if err := ledger.Record(now, &usage); err != nil {
		t.Fatal(err)
	}
	sent := ledger.Pending()[0]
	if err := ledger.Record(now, &usage); err != nil {
		t.Fatal(err)
	}
	ledger.Acked[sent.Date] = sent
	pending := ledger.Pending()
	if len(pending) != 1 || pending[0].Sessions != 2 || pending[0].Words != 240 {
		t.Fatalf("lost newer usage: %+v", pending)
	}
	if err := ledger.Record(now.AddDate(0, 0, 31), nil); err != nil {
		t.Fatal(err)
	}
	if len(ledger.Days) != 1 || len(ledger.Acked) != 0 {
		t.Fatalf("old data retained: %+v", ledger)
	}
}

func TestCalendarDayValidation(t *testing.T) {
	for _, date := range []string{"2026-02-30", "2026-13-01", "2026-1-01"} {
		if (Day{Date: date, Visited: true}).Valid() {
			t.Fatalf("invalid day %s", date)
		}
	}
}
