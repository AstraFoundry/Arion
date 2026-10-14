package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// fakeField is a followed field whose text and focus a test scripts.
type fakeField struct {
	mu       sync.Mutex
	text     string
	focused  bool
	hang     bool
	refuse   bool
	followed atomic.Int32
	released atomic.Int32
	reads    atomic.Int32
}

func (f *fakeField) Follow(contracts.TargetToken) bool {
	if f.refuse {
		return false
	}
	f.followed.Add(1)
	return true
}

func (f *fakeField) ReadField(ctx context.Context, _ contracts.TargetToken) (ports.FieldRead, error) {
	f.reads.Add(1)
	f.mu.Lock()
	hang, read := f.hang, ports.FieldRead{Text: f.text, Focused: f.focused}
	f.mu.Unlock()
	if hang {
		<-ctx.Done()
		time.Sleep(time.Second) // the host keeps hanging after the budget
		return ports.FieldRead{}, errors.New("hung")
	}
	return read, nil
}

func (f *fakeField) Release(contracts.TargetToken) { f.released.Add(1) }

func (f *fakeField) set(text string, focused bool) {
	f.mu.Lock()
	f.text, f.focused = text, focused
	f.mu.Unlock()
}

func newTracker(reports chan EditReport) *EditTracker {
	return &EditTracker{Poll: 5 * time.Millisecond, Idle: 60 * time.Millisecond, Max: 2 * time.Second,
		Report: func(r EditReport) { reports <- r }}
}

func awaitReport(t *testing.T, reports chan EditReport) EditReport {
	t.Helper()
	select {
	case r := <-reports:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("no report")
		return EditReport{}
	}
}

func TestEditTrackerLearnsAfterIdle(t *testing.T) {
	reports := make(chan EditReport, 1)
	tracker := newTracker(reports)
	field := &fakeField{text: "Hi, ask jason about it", focused: true}
	tracker.Track(field, "t", contracts.Identity{SessionID: "s"}, "ask jason about it")
	time.Sleep(20 * time.Millisecond)
	field.set("Hi, ask Jayson about it", true)
	r := awaitReport(t, reports)
	if r.Settlement != SettleIdle || r.Outcome.Verdict != domain.EditLearned || r.Outcome.Pairs[0] != (domain.Replacement{From: "jason", To: "Jayson"}) {
		t.Fatalf("report %+v", r)
	}
	if field.released.Load() != 1 {
		t.Fatal("field not released")
	}
}

// A chat app clears the field when the message is sent: the last text
// seen before the clear is judged.
func TestEditTrackerJudgesLastTextWhenSent(t *testing.T) {
	reports := make(chan EditReport, 1)
	tracker := newTracker(reports)
	tracker.Idle = time.Hour
	field := &fakeField{text: "打开杰森文件", focused: true}
	tracker.Track(field, "t", contracts.Identity{}, "打开杰森文件")
	time.Sleep(20 * time.Millisecond)
	field.set("打开JSON文件", true)
	time.Sleep(20 * time.Millisecond)
	field.set("", true)
	r := awaitReport(t, reports)
	if r.Settlement != SettleCleared || r.Outcome.Verdict != domain.EditLearned || r.Outcome.Pairs[0].To != "JSON" {
		t.Fatalf("report %+v", r)
	}
}

func TestEditTrackerSettlesOnFocusLoss(t *testing.T) {
	reports := make(chan EditReport, 1)
	tracker := newTracker(reports)
	tracker.Idle = time.Hour
	field := &fakeField{text: "明天开会", focused: true}
	tracker.Track(field, "t", contracts.Identity{}, "明天开会")
	time.Sleep(20 * time.Millisecond)
	field.set("明天开会", false)
	if r := awaitReport(t, reports); r.Settlement != SettleFocus || r.Outcome.Verdict != domain.EditUnchanged {
		t.Fatalf("report %+v", r)
	}
}

// A hung host costs at most the read budget per poll and is abandoned;
// Track itself never blocks.
func TestEditTrackerAbandonsHungHost(t *testing.T) {
	reports := make(chan EditReport, 1)
	tracker := newTracker(reports)
	field := &fakeField{hang: true}
	start := time.Now()
	tracker.Track(field, "t", contracts.Identity{}, "x")
	if time.Since(start) > 10*time.Millisecond {
		t.Fatal("Track blocked")
	}
	r := awaitReport(t, reports)
	if r.Settlement != SettleAbandon || r.Failures != editReadFailures || r.Outcome.Verdict != "" {
		t.Fatalf("report %+v", r)
	}
	if elapsed := time.Since(start); elapsed > editReadFailures*(EditReadBudget+50*time.Millisecond)+100*time.Millisecond {
		t.Fatalf("abandoning took %s", elapsed)
	}
}

func TestEditTrackerNextDictationSettles(t *testing.T) {
	reports := make(chan EditReport, 2)
	tracker := newTracker(reports)
	tracker.Idle = time.Hour
	field := &fakeField{text: "we use open ai models", focused: true}
	tracker.Track(field, "t", contracts.Identity{}, "we use open ai models")
	time.Sleep(20 * time.Millisecond)
	field.set("we use OpenAI models", true)
	time.Sleep(20 * time.Millisecond)
	start := time.Now()
	tracker.Settle()
	if time.Since(start) > 5*time.Millisecond {
		t.Fatal("Settle blocked")
	}
	if r := awaitReport(t, reports); r.Settlement != SettleNext || r.Outcome.Pairs[0].To != "OpenAI" {
		t.Fatalf("report %+v", r)
	}
}

func TestEditTrackerWithoutBaseline(t *testing.T) {
	reports := make(chan EditReport, 1)
	tracker := newTracker(reports)
	field := &fakeField{text: "something else", focused: true}
	tracker.Track(field, "t", contracts.Identity{}, "inserted")
	if r := awaitReport(t, reports); r.Settlement != SettleBaseline || r.Outcome.Verdict != "" {
		t.Fatalf("report %+v", r)
	}
}

// fieldTarget is a commit target that can be followed.
type fieldTarget struct {
	*fakeTarget
	*fakeField
}

func TestCoordinatorFollowsCommittedField(t *testing.T) {
	reports := make(chan EditReport, 1)
	tracker := newTracker(reports)
	field := &fakeField{text: "x hello", focused: true}
	target := fieldTarget{&fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "t1"}}, field}
	learning := atomic.Bool{}
	learning.Store(true)
	audio := &fakeAudio{}
	c, err := NewCoordinator(Dependencies{Audio: audio, Recognizers: fakeFactory{&fakeRecognizer{words: []string{"hello"}}},
		NewID: newID, Targets: target, Input: target, Edits: tracker}, Options{AutoCommit: true, Learning: learning.Load})
	if err != nil {
		t.Fatal(err)
	}
	events := &recorder{}
	c.Subscribe(events.listen)
	c.Toggle()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Toggle()
	events.wait(t, kindIs(contracts.CommitResultReady))
	if r := awaitReport(t, reports); r.Settlement != SettleIdle || r.Outcome.Verdict != domain.EditUnchanged {
		t.Fatalf("report %+v", r)
	}
	if field.followed.Load() != 1 || field.released.Load() != 1 {
		t.Fatalf("followed %d released %d", field.followed.Load(), field.released.Load())
	}
}

func TestCoordinatorNeverFollowsSensitiveOrWhenOff(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sensitive bool
		learning  bool
	}{{"sensitive", true, true}, {"off", false, false}} {
		field := &fakeField{text: "hello", focused: true}
		target := fieldTarget{&fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "t1", Sensitive: tc.sensitive}}, field}
		audio := &fakeAudio{}
		c, _ := NewCoordinator(Dependencies{Audio: audio, Recognizers: fakeFactory{&fakeRecognizer{words: []string{"hello"}}},
			NewID: newID, Targets: target, Input: target, Edits: newTracker(make(chan EditReport, 1))},
			Options{AutoCommit: true, Learning: func() bool { return tc.learning }})
		events := &recorder{}
		c.Subscribe(events.listen)
		c.Toggle()
		if !tc.sensitive {
			// A secure field fails before capture starts.
			events.wait(t, stateIs(domain.Capturing))
			audio.push(1, 0.2)
			c.Toggle()
		}
		events.wait(t, func(e contracts.Event) bool { return e.State == domain.Completed || e.State == domain.Failed })
		time.Sleep(30 * time.Millisecond)
		if field.followed.Load() != 0 || field.reads.Load() != 0 {
			t.Fatalf("%s: field followed %d read %d", tc.name, field.followed.Load(), field.reads.Load())
		}
	}
}
