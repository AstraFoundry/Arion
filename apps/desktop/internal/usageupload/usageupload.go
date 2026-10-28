// Package usageupload owns signed-in daily usage snapshots for AstraFoundry.
// Days hold visit and dictation totals only, never text, apps or hosts.
package usageupload

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"arion.local/core/domain"
)

const Product = "arion"
const MaxDays = 31

var (
	ErrPayload       = errors.New("usage_upload_payload")
	ErrAuthorization = errors.New("usage_upload_authorization")
	ErrHTTP          = errors.New("usage_upload_http")
	ErrStore         = errors.New("usage_upload_store")
	dayPattern       = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

type Session struct{ Issuer, Subject, Token string }

func (s Session) Valid() bool {
	return s.Issuer != "" && s.Subject != "" && len(s.Subject) <= 512 && s.Token != "" && !strings.ContainsAny(s.Token, " \t\r\n")
}

func (s Session) Key() string {
	sum := sha256.Sum256([]byte(s.Issuer + "\x00" + s.Subject + "\x00" + Product))
	return hex.EncodeToString(sum[:])
}

type Day struct {
	Date           string `json:"date"`
	Visited        bool   `json:"visited"`
	Sessions       int    `json:"sessions"`
	SpeechMs       int64  `json:"speechMs"`
	Words          int    `json:"words"`
	WordsPerMinute int    `json:"wordsPerMinute,omitempty"`
}

func (d Day) Valid() bool {
	if !dayPattern.MatchString(d.Date) || d.Sessions < 0 || d.Sessions > 10000 || d.SpeechMs < 0 || d.SpeechMs > 86_400_000 || d.Words < 0 || d.Words > 1_000_000 {
		return false
	}
	return d.Visited || (d.Sessions == 0 && d.SpeechMs == 0 && d.Words == 0)
}

func Same(a, b Day) bool {
	return a.Date == b.Date && a.Visited == b.Visited && a.Sessions == b.Sessions && a.SpeechMs == b.SpeechMs && a.Words == b.Words
}

func Snapshot(log domain.UsageLog, visited []string, today string) []Day {
	known := map[string]Day{}
	for _, day := range log {
		if !keep(day.Date, today) {
			continue
		}
		known[day.Date] = Day{
			Date:     day.Date,
			Visited:  true,
			Sessions: day.Sessions,
			SpeechMs: day.SpeechMs,
			Words:    day.Words,
		}
	}
	for _, date := range visited {
		if !keep(date, today) {
			continue
		}
		if current, ok := known[date]; ok {
			current.Visited = true
			known[date] = current
			continue
		}
		known[date] = Day{Date: date, Visited: true}
	}
	out := make([]Day, 0, len(known))
	for _, day := range known {
		out = append(out, day)
	}
	slices.SortFunc(out, func(a, b Day) int { return strings.Compare(a.Date, b.Date) })
	return out
}

func Pending(current []Day, acked map[string]Day) []Day {
	var out []Day
	for _, day := range current {
		if !day.Valid() {
			continue
		}
		if previous, ok := acked[day.Date]; ok && Same(previous, day) {
			continue
		}
		out = append(out, day)
	}
	return out
}

func keep(date, today string) bool {
	if !dayPattern.MatchString(date) || !dayPattern.MatchString(today) {
		return false
	}
	start, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return false
	}
	end, err := time.Parse(time.DateOnly, today)
	if err != nil {
		return false
	}
	return !start.After(end) && end.Sub(start) < time.Duration(MaxDays)*24*time.Hour
}
