package bootstrap

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/clipboardhistory"
	"arion.local/desktop/infrastructure/clipboardos"
	"arion.local/desktop/infrastructure/secrets"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

type clipboardController struct {
	previewWindow     *mygo.Window
	previewID         string
	previewAnchor     mygo.Rectangle
	previewHeight     int
	previewSeq        uint64
	previewLoading    bool
	anchor            mygo.Rectangle
	pendingDictations []domain.Clip
	service           *application.Clipboard
	capture           application.ClipboardCapture
	shortcut          string
	target            clipboardos.Target
	stop              chan struct{}
	lastPruned        time.Time
	pasteSeq          uint64
	windowSeq         uint64
	delivering        bool
	pollPending       atomic.Bool
	collecting        atomic.Bool
	observed          atomic.Uint64
}

func (a *App) startClipboard() {
	a.clipboard = &clipboardController{stop: make(chan struct{}), lastPruned: time.Now()}
	revision, _ := clipboardos.Revision()
	a.clipboard.capture.Configure(a.state.Preferences.Clipboard.Enabled, revision)
	a.clipboard.observed.Store(revision)
	a.clipboard.collecting.Store(a.state.Preferences.Clipboard.Enabled)
	a.registerClipboardShortcut()
	a.loadClipboard()
	mygo.App.OnQuit(func() {
		close(a.clipboard.stop)
		if a.clipboard.target != nil {
			a.clipboard.target.Close()
		}
	})
	go func() {
		maintenance := time.Now()
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-a.clipboard.stop:
				return
			case <-ticker.C:
				due := time.Since(maintenance) >= time.Minute
				if !due && (!a.clipboard.collecting.Load() || clipboardos.Sequence() == a.clipboard.observed.Load()) {
					continue
				}
				if a.clipboard.pollPending.CompareAndSwap(false, true) {
					if due {
						maintenance = time.Now()
					}
					a.post(func() { defer a.clipboard.pollPending.Store(false); a.pollClipboard() })
				}
			}
		}
	}()
}

func (a *App) configureClipboard(previous application.ClipboardPreferences) {
	if a.clipboard == nil {
		return
	}
	p := a.state.Preferences.Clipboard
	active := p.Enabled
	if !active {
		a.clipboard.pendingDictations = nil
	}
	revision, _ := clipboardos.Revision()
	a.clipboard.capture.Configure(active, revision)
	a.clipboard.collecting.Store(active)
	a.clipboard.observed.Store(revision)
	if p.Shortcut != previous.Shortcut {
		a.registerClipboardShortcut()
	}
	if p.Enabled && a.clipboard.service == nil && !a.state.Clipboard.Busy {
		a.loadClipboard()
	}
}

func (a *App) registerClipboardShortcut() {
	cc := a.clipboard
	if cc.shortcut != "" {
		mygo.GlobalShortcut.Unregister(cc.shortcut)
		cc.shortcut = ""
	}
	if a.state.Hold.Recording != "" {
		return
	}
	p := a.state.Preferences
	key, ok := application.ClipboardShortcutKey(runtime.GOOS, p.Clipboard.Shortcut)
	a.state.Clipboard.ShortcutFailed = true
	if !ok || application.HoldKeysConflict(key, p.HoldKey) || application.HoldKeysConflict(key, p.TranslateKey) {
		return
	}
	accelerator := clipboardAccelerator(key)
	if err := mygo.GlobalShortcut.Register(accelerator, func() {
		if a.state.Hold.Recording == "" {
			actions{a}.OpenClipboard()
		}
	}); err != nil {
		return
	}
	cc.shortcut = accelerator
	a.state.Clipboard.ShortcutFailed = false
	if a.state.Clipboard.Error == copy.ClipboardShortcutFailed {
		a.state.Clipboard.Error = ""
	}
}

func (a *App) loadClipboard() {
	if a.state.Clipboard.Busy {
		return
	}
	path := filepath.Join(a.opts.DataDir, "clipboard.bin")
	_, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) && !a.state.Preferences.Clipboard.Enabled {
		return
	}
	a.clipboardWork(func() ([]domain.Clip, error) {
		var key []byte
		if errors.Is(err, os.ErrNotExist) {
			key, err = secrets.Key("Arion", "clipboard-key")
		} else if err == nil {
			key, err = secrets.ExistingKey("Arion", "clipboard-key")
		}
		if err != nil {
			return nil, err
		}
		repo, e := clipboardhistory.Open(path, key)
		if e != nil {
			return nil, e
		}
		service := application.NewClipboard(repo)
		items, e := refreshClipboard(service, a.preferences().Clipboard.Limits, time.Now())
		if e == nil {
			a.post(func() { a.clipboard.service = service })
		}
		return items, e
	}, copy.ClipboardLoadFailed)
}

// One in-flight job bounds memory and preserves mutation order. Views publish
// only after an authenticated snapshot has been committed successfully.
func (a *App) clipboardWork(work func() ([]domain.Clip, error), failure copy.Key) {
	if a.state.Clipboard.Busy || a.updateApplying() {
		return
	}
	a.state.Clipboard.Busy = true
	cached := maps.Clone(a.state.Clipboard.Meta)
	a.resourceJobs.Add(1)
	go func() {
		defer a.resourceJobs.Add(-1)
		items, err := work()
		var previewErr error
		var meta map[string]arion.ClipboardMeta
		if err == nil {
			meta, previewErr = clipboardMetadata(items, cached)
		}
		a.post(func() {
			defer a.drainClipboardDictations()
			a.state.Clipboard.Busy = false
			if errors.Is(err, transfer.ErrFormat) {
				return
			}
			if err != nil {
				a.state.Clipboard.Error = clipboardError(err, failure)
				return
			}
			a.state.Clipboard.Items = eligibleClipboardItems(items, meta)
			a.state.Clipboard.Meta = meta
			a.positionClipboard()
			switch a.state.Clipboard.Error {
			case copy.ClipboardLoadFailed, copy.ClipboardSaveFailed, copy.ClipboardCapacity, copy.ClipboardOversize, copy.ClipboardReadFailed:
				a.state.Clipboard.Error = ""
			}
			if previewErr != nil {
				a.state.Clipboard.Error = copy.ClipboardPreviewFailed
			}
		})
	}()
}

func clipboardError(err error, other copy.Key) copy.Key {
	switch {
	case errors.Is(err, domain.ErrClipOversize):
		return copy.ClipboardOversize
	case errors.Is(err, domain.ErrClipCapacity):
		return copy.ClipboardCapacity
	default:
		return other
	}
}

func (a *App) pollClipboard() {
	cc := a.clipboard
	if cc == nil || a.state.Clipboard.Busy || a.state.Clipboard.Using || cc.delivering || a.updateApplying() {
		return
	}
	a.drainClipboardDictations()
	if a.state.Clipboard.Busy {
		return
	}
	if cc.service != nil && time.Since(cc.lastPruned) >= time.Minute {
		cc.lastPruned = time.Now()
		limits := a.state.Preferences.Clipboard.Limits
		a.clipboardWork(func() ([]domain.Clip, error) { return refreshClipboard(cc.service, limits, time.Now()) }, copy.ClipboardSaveFailed)
		return
	}
	if !a.state.Preferences.Clipboard.Enabled || cc.service == nil {
		return
	}
	unlock, ok := clipboardos.TryLock()
	if !ok {
		return
	}
	defer unlock()
	rev, excluded := clipboardos.Revision()
	if !cc.capture.Pending(rev, excluded) {
		cc.observed.Store(rev)
		return
	}
	payload, err := clipboardos.Read(a.state.Preferences.Clipboard.Limits.ItemBytes)
	current, _ := clipboardos.Revision()
	if current != rev || errors.Is(err, mygo.ErrClipboardChanged) {
		return
	}
	cc.capture.Accept(rev)
	cc.observed.Store(rev)
	if errors.Is(err, transfer.ErrFormat) {
		return
	}
	if err != nil {
		a.state.Clipboard.Error = clipboardError(err, copy.ClipboardReadFailed)
		return
	}
	limits := a.state.Preferences.Clipboard.Limits
	a.clipboardWork(func() ([]domain.Clip, error) {
		var err error
		payload, err = clipboardos.FilesOnly(payload)
		if err != nil {
			return nil, err
		}
		if err := clipboardhistory.ValidateResource(payload); err != nil {
			return nil, err
		}
		if payload.Kind == domain.ClipImage {
			if _, err := clipBitmap(payload.PNG, 96); err != nil {
				return nil, err
			}
		}
		return cc.service.Capture(payload, limits, time.Now())
	}, copy.ClipboardSaveFailed)
}

// Unsupported legacy references cannot retain invisible permanent pins. They
// become ordinary records and expire through the configured retention policy.
func refreshClipboard(service *application.Clipboard, limits domain.ClipLimits, now time.Time) ([]domain.Clip, error) {
	return service.Update(func(items []domain.Clip) ([]domain.Clip, error) {
		for i, item := range items {
			if item.Pinned && errors.Is(clipboardos.ValidateFiles(item.Payload), transfer.ErrFormat) {
				items[i].Pinned = false
			}
		}
		return domain.PruneClips(items, limits, now)
	})
}
