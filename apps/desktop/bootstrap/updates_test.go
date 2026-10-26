package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// fakeFeed answers checks with found or err; installs report two progress
// steps and then installErr.
type fakeFeed struct {
	enabled    bool
	found      *release
	err        error
	installErr error
	checks     int
	include    bool
}

func (f *fakeFeed) Enabled() bool { return f.enabled }

func (f *fakeFeed) Check(_ context.Context, include bool) (*release, error) {
	f.checks++
	f.include = include
	return f.found, f.err
}

func (f *fakeFeed) release(version string) *release {
	return &release{Version: version, Notes: "notes", Prepare: func(_ context.Context, progress func(int64, int64)) (preparedRelease, error) {
		progress(5, 10)
		progress(10, 10)
		if f.installErr != nil {
			return nil, f.installErr
		}
		return &fakePrepared{version: version}, nil
	}}
}

// harnessUpdates runs posted callbacks synchronously on the test goroutine.
func harnessUpdates(feed updateSource) (*updates, *arion.Update, func()) {
	state := &arion.Update{}
	posted := make(chan func(), 16)
	u := newUpdates(feed, state, func(fn func()) { posted <- fn })
	drain := func() {
		for {
			select {
			case fn := <-posted:
				fn()
			case <-time.After(50 * time.Millisecond):
				return
			}
		}
	}
	return u, state, drain
}

func TestUpdateCheckAndInstall(t *testing.T) {
	feed := &fakeFeed{enabled: true}
	feed.found = feed.release("0.2.0")
	u, state, drain := harnessUpdates(feed)
	if state.Phase != arion.UpdateIdle {
		t.Fatalf("initial phase %s", state.Phase)
	}
	now := time.Now()
	u.check(now, false)
	if state.Phase != arion.UpdateChecking {
		t.Fatalf("checking phase %s", state.Phase)
	}
	drain()
	if state.Phase != arion.UpdateAvailable || state.Version != "0.2.0" || state.Checked.IsZero() {
		t.Fatalf("found: %+v", state)
	}
	u.install()
	drain()
	if state.Phase != arion.UpdateStaged || state.Downloaded != 10 || state.Total != 10 {
		t.Fatalf("installed: %+v", state)
	}
	// A version waiting for a restart is not checked again.
	if u.due(now.Add(2 * updateInterval)) {
		t.Fatal("check due while an update waits for a restart")
	}
}

func TestUpdateFailures(t *testing.T) {
	feed := &fakeFeed{enabled: true, err: errors.New("offline")}
	u, state, drain := harnessUpdates(feed)
	// An automatic check that fails keeps what the user saw.
	u.check(time.Now(), false)
	drain()
	if state.Phase != arion.UpdateIdle {
		t.Fatalf("automatic failure: %+v", state)
	}
	// One the user asked for says so.
	u.check(time.Now(), true)
	drain()
	if state.Phase != arion.UpdateFailed || state.Failed != arion.UpdateStepCheck {
		t.Fatalf("manual failure: %+v", state)
	}
	feed.err, feed.installErr = nil, errors.New("bad signature")
	feed.found = feed.release("0.2.0")
	u.check(time.Now(), true)
	drain()
	u.install()
	drain()
	if state.Phase != arion.UpdateFailed || state.Failed != arion.UpdateStepPrepare || state.Version != "0.2.0" {
		t.Fatalf("install failure: %+v", state)
	}
	// Retrying the install starts over.
	feed.installErr = nil
	u.install()
	drain()
	if state.Phase != arion.UpdateStaged {
		t.Fatalf("retry: %+v", state)
	}
}

func TestUpdateSchedule(t *testing.T) {
	feed := &fakeFeed{enabled: true}
	u, state, drain := harnessUpdates(feed)
	start := time.Now()
	if !u.due(start) {
		t.Fatal("first check not due at launch")
	}
	u.check(start, false)
	drain()
	if state.Phase != arion.UpdateCurrent {
		t.Fatalf("up to date: %+v", state)
	}
	if u.due(start.Add(updateInterval - time.Minute)) {
		t.Fatal("check due before the interval")
	}
	if !u.due(start.Add(updateInterval)) {
		t.Fatal("check not due after the interval")
	}
	// A failed attempt also waits for the interval, so offline Macs do not
	// ask on every poll.
	feed.err = errors.New("offline")
	u.check(start.Add(updateInterval), false)
	drain()
	if u.due(start.Add(updateInterval + updatePoll)) {
		t.Fatal("check due one poll after a failed attempt")
	}
}

func TestUpdatesUnsupported(t *testing.T) {
	_, state, _ := harnessUpdates(nil)
	if state.Phase != arion.UpdateUnsupported {
		t.Fatalf("no source: %s", state.Phase)
	}
	feed := &fakeFeed{enabled: true, err: mygo.ErrUpdatesDisabled}
	u, state, drain := harnessUpdates(feed)
	u.check(time.Now(), true)
	drain()
	if state.Phase != arion.UpdateUnsupported || u.due(time.Now().Add(2*updateInterval)) {
		t.Fatalf("disabled build: %+v", state)
	}
	_, state, _ = harnessUpdates(&fakeFeed{})
	if state.Phase != arion.UpdateUnsupported {
		t.Fatalf("disabled source: %s", state.Phase)
	}
}

func TestUpdateChannels(t *testing.T) {
	feed := &fakeFeed{enabled: true, err: mygo.ErrNoPublishedUpdates}
	u, state, drain := harnessUpdates(feed)
	u.check(time.Now(), true)
	drain()
	if state.Phase != arion.UpdateNoRelease || feed.include {
		t.Fatalf("empty stable channel: %+v", state)
	}
	feed.err = nil
	feed.found = feed.release("0.1.0-alpha.6")
	u.setPrereleases(true)
	u.check(time.Now(), true)
	drain()
	if !feed.include || state.Phase != arion.UpdateAvailable {
		t.Fatalf("preview channel: %+v", state)
	}
	u.setPrereleases(false)
	if u.found != nil || state.Phase != arion.UpdateIdle {
		t.Fatalf("old channel result retained: %+v", state)
	}
}

type delayedUpdateFeed struct {
	entered chan struct{}
	finish  chan struct{}
}

func (f *delayedUpdateFeed) Enabled() bool { return true }
func (f *delayedUpdateFeed) Check(context.Context, bool) (*release, error) {
	close(f.entered)
	<-f.finish
	return &release{Version: "0.2.0-alpha.1"}, nil
}
func TestUpdateChannelChangeDiscardsInFlightResult(t *testing.T) {
	feed := &delayedUpdateFeed{make(chan struct{}), make(chan struct{})}
	u, state, drain := harnessUpdates(feed)
	u.setPrereleases(true)
	u.check(time.Now(), true)
	<-feed.entered
	u.setPrereleases(false)
	close(feed.finish)
	drain()
	if state.Phase != arion.UpdateIdle || u.found != nil {
		t.Fatalf("stale result applied: %+v", state)
	}
}

type fakePrepared struct {
	version            string
	applied, discarded int
	err                error
}

func (p *fakePrepared) Version() string { return p.version }
func (p *fakePrepared) Path() string    { return "/owned/stage/record.json" }
func (p *fakePrepared) Apply() error    { p.applied++; return p.err }
func (p *fakePrepared) Discard() error  { p.discarded++; return nil }

func (p *fakePrepared) Applied() bool      { return p.applied > 0 && p.err == nil }
func (p *fakePrepared) Acknowledge() error { return nil }
