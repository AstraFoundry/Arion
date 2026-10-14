package application

import (
	"context"
	"strings"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// Edit tracking timings (ADR 014). Typeless watches a field for 15 s after
// the last edit; Doubao settles when the user sends. Arion does both: it
// settles after EditIdle without change, when the field loses focus or is
// cleared (a chat message was sent), when the next dictation starts, and
// at the latest after EditMaxTrack.
const (
	EditPoll       = 400 * time.Millisecond
	EditIdle       = 15 * time.Second
	EditMaxTrack   = 60 * time.Second
	EditReadBudget = 250 * time.Millisecond
	// editBaselineWait is how long the inserted text may take to appear in
	// the field (keystroke and paste delivery are asynchronous).
	editBaselineWait = 2 * time.Second
	// editReadFailures abandons a field that stopped answering.
	editReadFailures = 3
)

// EditSettlement is why a tracked edit was evaluated.
type EditSettlement string

const (
	SettleIdle     EditSettlement = "idle"
	SettleFocus    EditSettlement = "focus_lost"
	SettleCleared  EditSettlement = "cleared"
	SettleNext     EditSettlement = "next_dictation"
	SettleMax      EditSettlement = "max_time"
	SettleAbandon  EditSettlement = "abandoned"
	SettleBaseline EditSettlement = "no_baseline"
)

// EditReport is delivered once per tracked commit. Outcome is empty when
// the edit could not be evaluated (Settlement abandoned or no_baseline).
type EditReport struct {
	Identity   contracts.Identity
	Settlement EditSettlement
	Outcome    domain.EditOutcome
	// Reads and Failures are content-free tracking counts.
	Reads    int
	Failures int
}

// EditTracker follows at most one committed field at a time and reports
// the corrections the user made in it. It never blocks a commit: Track
// returns at once and every read is bounded by EditReadBudget.
type EditTracker struct {
	// Report receives each settled edit on the tracker's goroutine.
	Report func(EditReport)
	// Poll, Idle and Max override the timings for tests.
	Poll, Idle, Max time.Duration
	// Now overrides the clock for tests.
	Now func() time.Time

	mu      sync.Mutex
	current *tracking
}

type tracking struct {
	reader   ports.FieldReader
	token    contracts.TargetToken
	identity contracts.Identity
	inserted string
	settle   chan EditSettlement
	done     chan struct{}
}

func (t *EditTracker) timings() (time.Duration, time.Duration, time.Duration) {
	poll, idle, max := t.Poll, t.Idle, t.Max
	if poll <= 0 {
		poll = EditPoll
	}
	if idle <= 0 {
		idle = EditIdle
	}
	if max <= 0 {
		max = EditMaxTrack
	}
	return poll, idle, max
}

func (t *EditTracker) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

// Track starts following a field the reader already follows. A previous
// tracking settles first, as when the next dictation starts.
func (t *EditTracker) Track(reader ports.FieldReader, token contracts.TargetToken, id contracts.Identity, inserted string) {
	t.Settle()
	tr := &tracking{reader: reader, token: token, identity: id, inserted: inserted,
		settle: make(chan EditSettlement, 1), done: make(chan struct{})}
	t.mu.Lock()
	t.current = tr
	t.mu.Unlock()
	go t.run(tr)
}

// Settle asks the current tracking to evaluate the last text it read and
// returns without waiting, so starting a dictation never waits on a host.
func (t *EditTracker) Settle() {
	t.mu.Lock()
	tr := t.current
	t.current = nil
	t.mu.Unlock()
	if tr != nil {
		select {
		case tr.settle <- SettleNext:
		default:
		}
	}
}

// Wait blocks until no tracking is running, for tests and shutdown.
func (t *EditTracker) Wait() {
	t.mu.Lock()
	tr := t.current
	t.mu.Unlock()
	if tr != nil {
		<-tr.done
	}
}

func (t *EditTracker) run(tr *tracking) {
	defer close(tr.done)
	defer tr.reader.Release(tr.token)
	report := EditReport{Identity: tr.identity}
	defer func() {
		t.mu.Lock()
		if t.current == tr {
			t.current = nil
		}
		t.mu.Unlock()
		if t.Report != nil {
			t.Report(report)
		}
	}()
	poll, idle, max := t.timings()
	start := t.now()
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	var baseline, last string
	changed := start
	failures := 0
	for {
		var why EditSettlement
		select {
		case why = <-tr.settle:
		case <-ticker.C:
		}
		if why == "" {
			read, err := t.read(tr)
			report.Reads++
			now := t.now()
			switch {
			case err != nil:
				failures++
				report.Failures++
				if failures >= editReadFailures {
					why = SettleAbandon
				}
			case baseline == "":
				failures = 0
				if strings.Contains(read.Text, tr.inserted) {
					baseline, last, changed = read.Text, read.Text, now
				} else if !read.Focused || now.Sub(start) > editBaselineWait {
					why = SettleBaseline
				}
			case !read.Focused:
				why = SettleFocus
			case strings.TrimSpace(read.Text) == "":
				why = SettleCleared
			default:
				failures = 0
				if read.Text != last {
					last, changed = read.Text, now
				}
				switch {
				case now.Sub(changed) >= idle:
					why = SettleIdle
				case now.Sub(start) >= max:
					why = SettleMax
				}
			}
		}
		if why == "" {
			continue
		}
		report.Settlement = why
		if baseline != "" && why != SettleAbandon {
			// A cleared or unfocused field is judged by the last text seen
			// in it, as Typeless does when a message is sent.
			report.Outcome = domain.ExtractCorrections(domain.FieldEdit{Inserted: tr.inserted, Baseline: baseline, Settled: last})
		} else if why == SettleNext && baseline == "" {
			report.Settlement = SettleBaseline
		}
		return
	}
}

// read bounds one native read so a hung host never stalls tracking.
func (t *EditTracker) read(tr *tracking) (ports.FieldRead, error) {
	ctx, cancel := context.WithTimeout(context.Background(), EditReadBudget)
	defer cancel()
	type answer struct {
		read ports.FieldRead
		err  error
	}
	ch := make(chan answer, 1)
	go func() {
		read, err := tr.reader.ReadField(ctx, tr.token)
		ch <- answer{read, err}
	}()
	select {
	case a := <-ch:
		return a.read, a.err
	case <-ctx.Done():
		return ports.FieldRead{}, ctx.Err()
	}
}
