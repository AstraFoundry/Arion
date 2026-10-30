package usageupload

import (
	"testing"

	"arion.local/core/domain"
)

func TestSnapshotMarksVisitAndKeepsDictationTotals(t *testing.T) {
	log := domain.UsageLog{
		{Date: "2026-10-26", Sessions: 1, SpeechMs: 60000, Words: 80},
		{Date: "2026-10-27", Sessions: 2, SpeechMs: 120000, Words: 200},
	}
	days := Snapshot(log, []string{"2026-10-27", "2026-10-28"}, "2026-10-27")
	if len(days) != 2 {
		t.Fatalf("days=%d", len(days))
	}
	if days[0].Date != "2026-10-26" || !days[0].Visited || days[0].Words != 80 {
		t.Fatalf("older %+v", days[0])
	}
	if days[1].Date != "2026-10-27" || days[1].Sessions != 2 || !days[1].Visited {
		t.Fatalf("today %+v", days[1])
	}
}

func TestPendingSkipsAcknowledgedDays(t *testing.T) {
	current := []Day{{Date: "2026-10-27", Visited: true, Sessions: 2, Words: 20}}
	if got := Pending(current, map[string]Day{"2026-10-27": current[0]}); len(got) != 0 {
		t.Fatalf("acked pending=%v", got)
	}
	if got := Pending(current, nil); len(got) != 1 {
		t.Fatalf("new pending=%v", got)
	}
}

func TestVisitOnlyDayIsValid(t *testing.T) {
	if !(Day{Date: "2026-10-27", Visited: true}).Valid() {
		t.Fatal("visit")
	}
	if (Day{Date: "2026-10-27", Sessions: 1}).Valid() {
		t.Fatal("counts without visit")
	}
}
