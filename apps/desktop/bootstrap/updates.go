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
// mygo.Updater: the latest GitHub release of the repository in desktop.json,
// whose archive must carry a signature by the key built into the app.
type updateSource interface {
	Enabled() bool
	Check(ctx context.Context, includePrereleases bool) (*release, error)
}

// release is a newer version and how to install it in place of the app.
type release struct {
	Version string
	Notes   string
	Install func(ctx context.Context, progress func(downloaded, total int64)) error
}

type mygoUpdates struct{}

func (mygoUpdates) Enabled() bool { return mygo.Updater.Enabled() }

func (mygoUpdates) Check(ctx context.Context, includePrereleases bool) (*release, error) {
	found, err := mygo.Updater.CheckWithOptions(ctx, mygo.UpdateCheckOptions{IncludePrereleases: includePrereleases})
	if err != nil || found == nil {
		return nil, err
	}
	return &release{Version: found.Version, Notes: found.Notes, Install: found.Install}, nil
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
}

func newUpdates(source updateSource, state *arion.Update, post func(func())) *updates {
	u := &updates{source: source, state: state, post: post}
	state.Phase = arion.UpdateUnsupported
	if source != nil && source.Enabled() {
		state.Phase = arion.UpdateIdle
	}
	return u
}

// setPrereleases invalidates a result from the previous channel.
func (u *updates) setPrereleases(include bool) {
	if u.includePrereleases == include {
		return
	}
	u.includePrereleases = include
	u.generation++
	if u.cancel != nil {
		u.cancel()
		u.cancel = nil
	}
	u.found = nil
	u.attempt = time.Time{}
	if u.state.Phase != arion.UpdateUnsupported && u.state.Phase != arion.UpdateDownloading && u.state.Phase != arion.UpdateReady {
		*u.state = arion.Update{Phase: arion.UpdateIdle}
	}
}

// due reports whether an automatic check should run at now.
func (u *updates) due(now time.Time) bool {
	switch u.state.Phase {
	case arion.UpdateUnsupported, arion.UpdateChecking, arion.UpdateDownloading, arion.UpdateReady:
		return false
	}
	return u.attempt.IsZero() || now.Sub(u.attempt) >= updateInterval
}

// check asks the feed for a newer version. An automatic check that fails
// keeps what the user saw, since being offline is not news; a check the
// user asked for reports the failure.
func (u *updates) check(now time.Time, manual bool) {
	switch u.state.Phase {
	case arion.UpdateUnsupported, arion.UpdateChecking, arion.UpdateDownloading, arion.UpdateReady:
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

// install downloads the found version, checks its signature and replaces
// the app on disk; the running app is untouched until a restart.
func (u *updates) install() {
	if u.found == nil || (u.state.Phase != arion.UpdateAvailable && u.state.Phase != arion.UpdateFailed) {
		return
	}
	found := u.found
	u.state.Phase, u.state.Failed, u.state.Downloaded, u.state.Total = arion.UpdateDownloading, "", 0, 0
	go func() {
		err := found.Install(context.Background(), func(downloaded, total int64) {
			u.post(func() {
				if u.state.Phase == arion.UpdateDownloading {
					u.state.Downloaded, u.state.Total = downloaded, total
				}
			})
		})
		u.post(func() {
			if err != nil {
				if u.record != nil {
					u.record("update_install", err)
				}
				u.state.Phase, u.state.Failed = arion.UpdateFailed, arion.UpdateStepInstall
				return
			}
			if u.record != nil {
				u.record("update_install", nil)
			}
			u.state.Phase = arion.UpdateReady
		})
	}()
}

// scheduleUpdates runs automatic checks while the preference allows them.
func (a *App) scheduleUpdates() {
	a.updates.record = a.record
	tick := func() {
		if a.state.Preferences.CheckUpdates && a.updates.due(time.Now()) {
			a.updates.check(time.Now(), false)
		}
	}
	tick()
	go func() {
		for range time.Tick(updatePoll) {
			a.post(tick)
		}
	}()
}
