package domain

import (
	"testing"
	"time"
)

func TestCountWords(t *testing.T) {
	for text, want := range map[string]int{
		"":                         0,
		"Hello, world!":            2,
		"don't use e-mail at 3.5x": 5,
		"今天天气不错":                   6,
		"把 README 发给 Alice，谢谢。":    7,
		"こんにちは":                    5,
		"   \n\t":                  0,
		"v2 build-42 ready.":       3,
		"结束了.":                     3,
		"end.":                     1,
	} {
		if got := CountWords(text); got != want {
			t.Errorf("CountWords(%q) = %d, want %d", text, got, want)
		}
	}
}

func TestUsageLogAddsByDay(t *testing.T) {
	day := time.Date(2026, 10, 14, 9, 0, 0, 0, time.Local)
	var log UsageLog
	log = log.Add(day, Usage{SpeechMs: 30000, Words: 50})
	before := log
	log = log.Add(day.Add(3*time.Hour), Usage{SpeechMs: 30000, Words: 70, Scene: SceneEmail})
	log = log.Add(day.AddDate(0, 0, -2), Usage{SpeechMs: 6000, Words: 10, Scene: SceneEmail})
	if len(before) != 1 || before[0].Words != 50 || before[0].Scenes != nil {
		t.Fatalf("Add mutated its receiver: %+v", before)
	}
	if len(log) != 2 || log[0].Date != "2026-10-12" || log[1].Date != "2026-10-14" {
		t.Fatalf("days not ordered: %+v", log)
	}
	today := log[1]
	if today.Sessions != 2 || today.Words != 120 || today.SpeechMs != 60000 || today.PolishedWords() != 70 || today.WordsPerMinute() != 120 {
		t.Fatalf("today: %+v", today)
	}
	all := log.Since(time.Time{})
	if all.Words != 130 || all.Scenes[SceneEmail] != 80 || all.Sessions != 3 {
		t.Fatalf("all: %+v", all)
	}
	if got := log.Since(day.AddDate(0, 0, -1)); got.Words != 120 {
		t.Fatalf("since yesterday: %+v", got)
	}
	week := log.Days(day, 7)
	if len(week) != 7 || week[0].Date != "2026-10-08" || week[4].Words != 10 || week[5].Words != 0 || week[6].Words != 120 {
		t.Fatalf("week: %+v", week)
	}
	if (UsageDay{}).WordsPerMinute() != 0 {
		t.Fatal("speed without speech")
	}
}
