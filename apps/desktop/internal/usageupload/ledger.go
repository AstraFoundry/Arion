package usageupload

import (
	"arion.local/core/domain"
	"sort"
	"time"
)

// Ledger contains only usage committed while this account consented and was signed in.
// Source is random per installation/account; it is never a hardware identifier.
type Ledger struct {
	Version int            `json:"version"`
	Account string         `json:"account"`
	Source  string         `json:"source"`
	Days    map[string]Day `json:"days"`
	Acked   map[string]Day `json:"acked"`
}

func (l *Ledger) Record(at time.Time, usage *domain.Usage) error {
	if l.Days == nil {
		l.Days = map[string]Day{}
	}
	date := domain.DayKey(at)
	day := l.Days[date]
	day.Date, day.Visited = date, true
	if usage != nil {
		day.Sessions++
		day.SpeechMs += max(usage.SpeechMs, 0)
		day.Words += usage.Words
	}
	if !day.Valid() {
		return ErrPayload
	}
	l.Days[date] = day
	for date := range l.Days {
		if !keep(date, day.Date) {
			delete(l.Days, date)
			delete(l.Acked, date)
		}
	}
	return nil
}

func (l Ledger) Pending() []Day {
	days := make([]Day, 0, len(l.Days))
	for _, day := range l.Days {
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	return Pending(days, l.Acked)
}
