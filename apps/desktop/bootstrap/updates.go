package bootstrap

import (
	"context"
	"errors"
	"time"

	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// Update timing: the feed is asked at launch and again once updateInterval
// has passed since the last attempt. The clock is read every updatePoll,
// so a Mac that slept through a check catches up soon after waking.
const (
	updateInterval = 6 * time.Hour
	updatePoll     = 15 * time.Minute
	checkTimeout   = 30 * time.Second
)

// updateSource finds and installs newer versions. In production it is
// mygo.Updater: the channel manifest that the Arion center serves at the
// desktop.json updates.url, selected by the Allow beta updates switch,
// whose archive must carry a signature by the key built into the app.
type updateSource interface {
	Enabled() bool
	Check(ctx context.Context, includePrereleases bool) (*release, error)
}

// release is a newer version and how to install it in place of the app.
type release struct {
	Version string
	Notes   string
	Prepare func(ctx context.Context, progress func(downloaded, total int64)) (preparedRelease, error)
}

type preparedRelease interface {
	Apply() error
	Discard() error
	Path() string
	Version() string
	Applied() bool
	Acknowledge() error
}

type mygoUpdates struct{}

func (mygoUpdates) Enabled() bool { return mygo.Updater.Enabled() }

func (mygoUpdates) Check(ctx context.Context, includePrereleases bool) (*release, error) {
	found, err := mygo.Updater.CheckWithOptions(ctx, mygo.UpdateCheckOptions{IncludePrereleases: includePrereleases})
	if err != nil || found == nil {
		return nil, err
	}
	return &release{Version: found.Version, Notes: found.Notes, Prepare: func(ctx context.Context, progress func(int64, int64)) (preparedRelease, error) {
		return found.Prepare(ctx, progress)
	}}, nil
}

// updates drives the update state on the main thread; checks and installs
// run on their own goroutines and report through post.
type updates struct {
	source             updateSource
	state              *arion.Update
	post               func(func())
	found              *release
	attempt            time.Time
	includePrereleases bool
	generation         uint64
	cancel             context.CancelFunc
	record             func(string, error)
	prepared           preparedRelease
	automatic          bool
	autoCheck          bool
	autoInstall        bool
	manualCheck        bool
	recovering         bool
	stopped            bool
	blocked            bool
	persist            func(preparedRelease, bool) error
}

func newUpdates(source updateSource, state *arion.Update, post func(func())) *updates {
	if source != nil && !source.Enabled() {
		source = nil
	}
	u := &updates{source: source, state: state, post: post}
	state.Phase = arion.UpdateUnsupported
	if source != nil {
		state.Phase = arion.UpdateIdle
	}
	return u
}

// setPrereleases invalidates prepared data and every callback from the old channel.
func (u *updates) setPrereleases(include bool) {
	if u.includePrereleases == include || u.state.Phase == arion.UpdateApplying || u.state.Phase == arion.UpdateReady {
		return
	}
	if !u.invalidate() {
		return
	}
	u.includePrereleases = include
	u.found = nil
	u.attempt = time.Time{}
	if u.state.Phase != arion.UpdateUnsupported {
		*u.state = arion.Update{Phase: arion.UpdateIdle}
	}
}

func (u *updates) invalidate() bool {
	u.generation++
	u.automatic = false
	if u.cancel != nil {
		u.cancel()
		u.cancel = nil
	}
	u.state.RestartQueued = false
	if u.prepared != nil {
		if err := u.prepared.Discard(); err != nil {
			u.recoveryFailure(err)
			return false
		}
		u.prepared = nil
		if u.persist != nil {
			if err := u.persist(nil, false); err != nil {
				u.recoveryFailure(err)
				return false
			}
		}
	}
	return true
}

func (u *updates) log(event string, err error) {
	if u.record != nil {
		u.record(event, err)
	}
}

func (u *updates) preferences(check, install bool) {
	wasEnabled := u.autoCheck && u.autoInstall
	u.autoCheck, u.autoInstall = check, install
	if u.automatic && (!check || !install) && u.state.Phase != arion.UpdateApplying && u.state.Phase != arion.UpdateReady {
		if !u.invalidate() {
			return
		}
		if u.found != nil {
			u.state.Phase = arion.UpdateAvailable
		} else {
			u.state.Phase = arion.UpdateIdle
		}
	}
	if !check && !u.manualCheck && u.state.Phase == arion.UpdateChecking {
		if !u.invalidate() {
			return
		}
		u.state.Phase = arion.UpdateIdle
	}
	if !wasEnabled && check && install && u.state.Phase == arion.UpdateAvailable {
		u.prepare(true)
	}
}

// due reports whether an automatic check should run at now.
func (u *updates) due(now time.Time) bool {
	if u.source == nil || u.stopped || u.blocked || u.recovering || u.state.Failed == arion.UpdateStepRestart {
		return false
	}
	switch u.state.Phase {
	case arion.UpdateUnsupported, arion.UpdateChecking, arion.UpdateDownloading, arion.UpdateStaged, arion.UpdateApplying, arion.UpdateReady:
		return false
	}
	return u.attempt.IsZero() || now.Sub(u.attempt) >= updateInterval
}

// check asks the feed for a newer version. An automatic check that fails
// keeps what the user saw, since being offline is not news; a check the
// user asked for reports the failure.
func (u *updates) check(now time.Time, manual bool) {
	if u.source == nil || u.stopped || u.blocked || u.recovering || u.state.Failed == arion.UpdateStepRestart {
		return
	}
	u.manualCheck = manual
	switch u.state.Phase {
	case arion.UpdateUnsupported, arion.UpdateChecking, arion.UpdateDownloading, arion.UpdateStaged, arion.UpdateApplying, arion.UpdateReady:
		return
	}
	previous := *u.state
	u.attempt = now
	u.state.Phase = arion.UpdateChecking
	u.generation++
	generation, include := u.generation, u.includePrereleases
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	u.cancel = cancel
	go func() {
		defer cancel()
		found, err := u.source.Check(ctx, include)
		u.post(func() {
			if generation != u.generation {
				return
			}
			u.cancel = nil
			u.checked(found, err, previous, manual)
			if !manual && u.autoCheck && u.autoInstall && u.state.Phase == arion.UpdateAvailable {
				u.prepare(true)
			}
		})
	}()
}

func (u *updates) checked(found *release, err error, previous arion.Update, manual bool) {
	if errors.Is(err, mygo.ErrUpdatesDisabled) {
		*u.state = arion.Update{Phase: arion.UpdateUnsupported}
		return
	}
	if errors.Is(err, mygo.ErrNoPublishedUpdates) {
		u.found = nil
		*u.state = arion.Update{Phase: arion.UpdateNoRelease, Checked: time.Now()}
		if u.record != nil {
			u.record("update_check", nil)
		}
		return
	}
	if err != nil {
		if u.record != nil {
			u.record("update_check", err)
		}
		if !manual {
			*u.state = previous
			return
		}
		*u.state = arion.Update{Phase: arion.UpdateFailed, Failed: arion.UpdateStepCheck, Checked: previous.Checked,
			Version: previous.Version, Notes: previous.Notes}
		return
	}
	if u.record != nil {
		u.record("update_check", nil)
	}
	u.found = found
	*u.state = arion.Update{Phase: arion.UpdateCurrent, Checked: time.Now()}
	if found != nil {
		u.state.Phase, u.state.Version, u.state.Notes = arion.UpdateAvailable, found.Version, found.Notes
	}
}

// install prepares verified bytes; applying is reserved for the guarded restart.
func (u *updates) install() { u.prepare(false) }
func (u *updates) prepare(automatic bool) {
	if u.stopped || u.blocked || u.found == nil || u.found.Prepare == nil || u.prepared != nil || (u.state.Phase != arion.UpdateAvailable && u.state.Phase != arion.UpdateFailed) {
		return
	}
	found := u.found
	u.generation++
	gen := u.generation
	u.automatic = automatic
	u.state.Phase, u.state.Failed, u.state.Downloaded, u.state.Total = arion.UpdateDownloading, "", 0, 0
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	u.cancel = cancel
	go func() {
		defer cancel()
		prepared, err := found.Prepare(ctx, func(done, total int64) {
			u.post(func() {
				if gen == u.generation && u.state.Phase == arion.UpdateDownloading {
					u.state.Downloaded, u.state.Total = done, total
				}
			})
		})
		u.post(func() {
			if gen != u.generation || u.stopped {
				if prepared != nil {
					u.log("update_install", prepared.Discard())
				}
				return
			}
			u.cancel = nil
			if err == nil && prepared == nil {
				err = errors.New("updater returned no prepared artifact")
			}
			if err == nil && u.persist != nil {
				err = u.persist(prepared, automatic)
			}
			if err != nil {
				if prepared != nil {
					u.log("update_install", prepared.Discard())
				}
				u.log("update_install", err)
				u.state.Phase, u.state.Failed = arion.UpdateFailed, arion.UpdateStepPrepare
				return
			}
			u.prepared = prepared
			u.state.Phase = arion.UpdateStaged
			u.log("update_install", nil)
		})
	}()
}

// apply starts only while the caller owns the input/resource maintenance lease.
func (u *updates) apply(done func(error)) {
	if u.prepared == nil || (u.state.Phase != arion.UpdateStaged && u.state.Phase != arion.UpdateFailed) {
		done(errors.New("no staged update"))
		return
	}
	prepared := u.prepared
	u.state.Phase, u.state.Failed = arion.UpdateApplying, ""
	go func() {
		err := prepared.Apply()
		u.post(func() {
			if err != nil {
				u.log("update_install", err)
				if errors.Is(err, mygo.ErrUpdateRecoveryRequired) {
					u.recoveryFailure(err)
				} else {
					discardErr := prepared.Discard()
					if discardErr == nil && u.persist != nil {
						discardErr = u.persist(nil, false)
					}
					if discardErr != nil {
						err = errors.Join(err, mygo.ErrUpdateRecoveryRequired, discardErr)
						u.recoveryFailure(err)
					} else {
						u.prepared = nil
						u.state.Phase, u.state.Failed = arion.UpdateFailed, arion.UpdateStepPrepare
					}
				}
				done(err)
				return
			}
			completionErr := u.acknowledgeApplied(prepared, u.automatic)
			if completionErr != nil {
				u.log("update_install", completionErr)
				if errors.Is(completionErr, mygo.ErrUpdateRecoveryRequired) {
					u.recoveryFailure(completionErr)
					done(completionErr)
					return
				}
			}
			u.state.Phase = arion.UpdateReady
			done(nil)
		})
	}()
}

func (u *updates) close() {
	u.stopped = true
	u.generation++
	if u.cancel != nil {
		u.cancel()
		u.cancel = nil
	}
}

func (u *updates) recoveryFailure(err error) {
	u.blocked = true
	u.state.Phase, u.state.Failed = arion.UpdateFailed, arion.UpdateStepRecovery
	u.state.RestartQueued = false
	u.log("update_install", err)
}

// Keep a completed receipt until both the application handle and updater
// journal have been reconciled. A failed acknowledgement must restore the
// durable handle before the old process can safely leave recovery to startup.
func (u *updates) acknowledgeApplied(prepared preparedRelease, automatic bool) error {
	u.prepared = prepared
	if u.persist != nil {
		if err := u.persist(nil, false); err != nil {
			u.state.StateError = true
			return err
		}
	}
	if err := prepared.Acknowledge(); err != nil {
		var restoreErr error
		if u.persist != nil {
			restoreErr = u.persist(prepared, automatic)
		} else {
			restoreErr = errors.New("completed receipt persistence is unavailable")
		}
		u.state.StateError = true
		if restoreErr != nil {
			return errors.Join(mygo.ErrUpdateRecoveryRequired, err, restoreErr)
		}
		return err
	}
	u.prepared = nil
	return nil
}
