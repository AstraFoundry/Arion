package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arion.local/desktop/infrastructure/updatestate"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

func TestAutomaticUpdateRequiresBothSwitchesAndAutomaticCheck(t *testing.T) {
	for _, tc := range []struct{ check, install, manual, want bool }{{true, true, false, true}, {true, false, false, false}, {false, true, false, false}, {true, true, true, false}} {
		f := &fakeFeed{enabled: true}
		f.found = f.release("1.2.3")
		u, s, drain := harnessUpdates(f)
		u.preferences(tc.check, tc.install)
		u.check(time.Now(), tc.manual)
		drain()
		if (s.Phase == arion.UpdateStaged) != tc.want {
			t.Fatalf("%+v -> %s", tc, s.Phase)
		}
	}
}
func TestDisableAutomaticDownloadRejectsLateProgressAndResult(t *testing.T) {
	entered, finish := make(chan struct{}), make(chan struct{})
	stage := &fakePrepared{version: "1.2.3"}
	f := &fakeFeed{enabled: true, found: &release{Version: "1.2.3", Prepare: func(ctx context.Context, p func(int64, int64)) (preparedRelease, error) {
		close(entered)
		<-finish
		p(10, 10)
		return stage, nil
	}}}
	u, s, drain := harnessUpdates(f)
	u.preferences(true, true)
	u.check(time.Now(), false)
	drain()
	<-entered
	u.preferences(true, false)
	close(finish)
	drain()
	if s.Phase != arion.UpdateAvailable || s.Downloaded != 0 || u.prepared != nil || stage.discarded != 1 {
		t.Fatalf("late download applied: %+v stage=%+v", s, stage)
	}
}
func TestChannelChangeDiscardsStagedArtifact(t *testing.T) {
	f := &fakeFeed{enabled: true}
	f.found = f.release("1.2.3")
	u, s, drain := harnessUpdates(f)
	u.check(time.Now(), true)
	drain()
	u.install()
	drain()
	stage := u.prepared.(*fakePrepared)
	u.setPrereleases(true)
	if u.prepared != nil || stage.discarded != 1 || s.Phase != arion.UpdateIdle {
		t.Fatalf("stale stage retained: %+v", s)
	}
}
func TestStageMustPersistBeforeItBecomesReady(t *testing.T) {
	f := &fakeFeed{enabled: true}
	f.found = f.release("1.2.3")
	u, s, drain := harnessUpdates(f)
	u.persist = func(preparedRelease, bool) error { return errors.New("disk full") }
	u.check(time.Now(), true)
	drain()
	u.install()
	drain()
	if s.Phase != arion.UpdateFailed || s.Failed != arion.UpdateStepPrepare || u.prepared != nil {
		t.Fatalf("unpersisted stage offered: %+v", s)
	}
}
func TestApplyIsSeparateFromDownloadAndRetryable(t *testing.T) {
	f := &fakeFeed{enabled: true}
	f.found = f.release("1.2.3")
	u, s, drain := harnessUpdates(f)
	u.check(time.Now(), true)
	drain()
	u.install()
	drain()
	stage := u.prepared.(*fakePrepared)
	if stage.applied != 0 {
		t.Fatal("download applied the artifact")
	}
	stage.err = errors.New("locked")
	var result error
	u.apply(func(err error) { result = err })
	drain()
	if result == nil || s.Failed != arion.UpdateStepPrepare || u.prepared != nil {
		t.Fatalf("failed swap lost retry: %+v", s)
	}
	u.install()
	drain()
	stage = u.prepared.(*fakePrepared)
	u.apply(func(err error) { result = err })
	drain()
	if result != nil || s.Phase != arion.UpdateReady || u.prepared != nil || stage.applied != 1 {
		t.Fatalf("retry failed: %+v", s)
	}
}
func TestReminderLaterAndDismissPersistAcrossAppLifetime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "updates.json")
	a := &App{updatePolicy: &updatePolicy{store: updatestate.Store{Path: path}, saved: updatestate.State{Schema: 1}}}
	a.state.Update = arion.Update{Phase: arion.UpdateAvailable, Version: "1.2.3"}
	key := a.reminderKey()
	later := time.Now().Add(updateLater)
	if err := a.saveReminder(key, func(r *updatestate.Reminder) { r.Later = later }); err != nil {
		t.Fatal(err)
	}
	loaded, err := a.updatePolicy.store.Load()
	if err != nil || !loaded.Reminders[0].Later.Equal(later) {
		t.Fatalf("later not persisted: %+v %v", loaded, err)
	}
	a.updatePolicy.saved = loaded
	if err = a.saveReminder(key, func(r *updatestate.Reminder) { r.Dismissed = true }); err != nil {
		t.Fatal(err)
	}
	loaded, err = a.updatePolicy.store.Load()
	if err != nil || !loaded.Reminders[0].Dismissed || len(loaded.Reminders) != 1 {
		t.Fatalf("dismiss: %+v %v", loaded, err)
	}
	a.state.Preferences.AllowPrereleaseUpdates = true
	if a.reminderKey() == key {
		t.Fatal("channel not part of reminder identity")
	}
	a.state.Preferences.AllowPrereleaseUpdates = false
	a.state.Update.Phase = arion.UpdateStaged
	if a.reminderKey() == key {
		t.Fatal("download completion shares availability identity")
	}
}

func TestFailedRollbackBlocksFurtherUpdatesAndPreservesJournal(t *testing.T) {
	f := &fakeFeed{enabled: true}
	f.found = f.release("1.2.3")
	u, s, drain := harnessUpdates(f)
	u.check(time.Now(), true)
	drain()
	u.install()
	drain()
	stage := u.prepared.(*fakePrepared)
	stage.err = mygo.ErrUpdateRecoveryRequired
	u.apply(func(error) {})
	drain()
	if s.Failed != arion.UpdateStepRecovery || !u.blocked || stage.discarded != 0 || u.prepared != stage {
		t.Fatalf("recovery bytes not preserved: %+v", s)
	}
	u.check(time.Now(), true)
	u.install()
	if s.Phase != arion.UpdateFailed || u.due(time.Now().Add(updateInterval)) {
		t.Fatal("blocked transaction restarted")
	}
}

func TestUnsupportedBuildDoesNotRecoverOrRewriteSavedUpdate(t *testing.T) {
	for _, content := range []string{`{"schema":1,"stage":"/release/update-stage","version":"9.9.9"}`, `invalid state`} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "updates.json")
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			a := &App{opts: Options{DataDir: dir}}
			a.updates = newUpdates(&fakeFeed{enabled: false}, &a.state.Update, nil)
			a.scheduleUpdates()
			if a.updatePolicy != nil || a.updates.recovering || a.updates.blocked || a.state.Update.Phase != arion.UpdateUnsupported || a.state.Update.StateError {
				t.Fatalf("unsupported build entered update recovery: %+v", a.state.Update)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != content {
				t.Fatalf("saved release transaction modified: %q %v", got, err)
			}
		})
	}
}
