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
	accessWindow      *mygo.Window
	filesGeneration   atomic.Uint64
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
	pasteSeq          uint64
	windowSeq         uint64
	delivering        bool
	pollPending       atomic.Bool
	collecting        atomic.Bool
	observed          atomic.Uint64
}

func (a *App) startClipboard() {
	a.clipboard = &clipboardController{stop: make(chan struct{})}
	revision, _ := clipboardos.Revision()
	a.clipboard.capture.Configure(a.state.Preferences.Clipboard.Enabled, revision)
	a.clipboard.observed.Store(revision)
	a.clipboard.collecting.Store(a.state.Preferences.Clipboard.Enabled)
	a.registerClipboardShortcut()
	a.loadClipboard()
	mygo.App.OnQuit(func() {
		a.dismissClipboardAccess()
		close(a.clipboard.stop)
		if a.clipboard.target != nil {
			a.clipboard.target.Close()
		}
	})
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-a.clipboard.stop:
				return
			case <-ticker.C:
				if !a.clipboard.collecting.Load() || clipboardos.Sequence() == a.clipboard.observed.Load() {
					continue
				}
				if a.clipboard.pollPending.CompareAndSwap(false, true) {
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
		a.dismissClipboardAccess()
		a.clipboard.pendingDictations = nil
		a.clipboard.windowSeq++
		a.clipboard.pasteSeq++
		actions{a}.CloseClipboard()
		a.dismissClipboardPreview()
	}
	if p.FilesEnabled != previous.FilesEnabled {
		a.dismissClipboardAccess()
		a.clipboard.filesGeneration.Add(1)
		a.clipboard.windowSeq++
		a.clipboard.pasteSeq++
		actions{a}.CloseClipboard()
		a.dismissClipboardPreview()
		a.state.Clipboard.Items = p.VisibleItems(a.state.Clipboard.Items)
		if !p.FilesEnabled {
			for id, meta := range a.state.Clipboard.Meta {
				if len(meta.Files) != 0 {
					delete(a.state.Clipboard.Meta, id)
				}
			}
		}
		if p.Enabled && !a.state.Clipboard.Busy && a.clipboard.service != nil {
			limits := p.Limits
			a.clipboardWork(func() ([]domain.Clip, error) { return refreshClipboard(a.clipboard.service, limits, p.FilesEnabled) }, copy.ClipboardLoadFailed)
		}
	}
	revision, _ := clipboardos.Revision()
	if p.FilesEnabled != previous.FilesEnabled {
		a.clipboard.capture.Accept(revision)
	}
	a.clipboard.capture.Configure(active, revision)
	a.clipboard.collecting.Store(active)
	a.clipboard.observed.Store(revision)
	if p.Enabled != previous.Enabled || p.Shortcut != previous.Shortcut {
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
	if !a.state.Preferences.Clipboard.Enabled {
		a.state.Clipboard.ShortcutFailed = false
		return
	}
	if a.state.Hold.Recording != "" {
		return
	}
	p := a.state.Preferences
	key, ok := application.ClipboardShortcutKey(runtime.GOOS, p.Clipboard.Shortcut)
	a.state.Clipboard.ShortcutFailed = true
	if !ok || application.HoldKeysConflict(key, p.HoldKey) || application.HoldKeysConflict(key, p.SecondHoldKey) || application.HoldKeysConflict(key, p.TranslateKey) {
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
			key, err = secrets.Key(desktopProfile(mygo.IsDev()), "clipboard-key")
		} else if err == nil {
			key, err = secrets.ExistingKey(desktopProfile(mygo.IsDev()), "clipboard-key")
		}
		if err != nil {
			return nil, err
		}
		repo, e := clipboardhistory.Open(path, key)
		if e != nil {
			return nil, e
		}
		service := application.NewClipboard(repo)
		items, e := refreshClipboard(service, a.preferences().Clipboard.Limits, a.preferences().Clipboard.FilesEnabled)
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
	filesGeneration := a.clipboard.filesGeneration.Load()
	a.resourceJobs.Add(1)
	go func() {
		defer a.resourceJobs.Add(-1)
		items, err := work()
		var meta map[string]arion.ClipboardMeta
		if err == nil {
			meta = clipboardMetadata(a.preferences().Clipboard.VisibleItems(items), cached)
		}
		a.post(func() {
			defer a.drainClipboardDictations()
			a.state.Clipboard.Busy = false
			if err != nil && !errors.Is(err, transfer.ErrFormat) {
				a.state.Clipboard.Error = clipboardError(err, failure)
				return
			}
			// A mode change invalidates metadata prepared under the previous policy.
			// Re-read the serialized snapshot instead of publishing incomplete file rows.
			if filesGeneration != a.clipboard.filesGeneration.Load() && a.state.Preferences.Clipboard.Enabled && a.clipboard.service != nil {
				prefs := a.state.Preferences.Clipboard
				a.clipboardWork(func() ([]domain.Clip, error) {
					return refreshClipboard(a.clipboard.service, prefs.Limits, prefs.FilesEnabled)
				}, copy.ClipboardLoadFailed)
				return
			}
			if errors.Is(err, transfer.ErrFormat) {
				return
			}
			a.state.Clipboard.Items = eligibleClipboardItems(a.state.Preferences.Clipboard.VisibleItems(items), meta)
			visibleMeta := make(map[string]arion.ClipboardMeta, len(a.state.Clipboard.Items))
			for _, item := range a.state.Clipboard.Items {
				visibleMeta[item.ID] = meta[item.ID]
			}
			a.state.Clipboard.Meta = visibleMeta
			a.positionClipboard()
			switch a.state.Clipboard.Error {
			case copy.ClipboardLoadFailed, copy.ClipboardSaveFailed, copy.ClipboardCapacity, copy.ClipboardOversize, copy.ClipboardReadFailed, copy.ClipboardAccessRequired, copy.ClipboardAccessReCopy:
				a.state.Clipboard.Error = ""
			}
		})
	}()
}

func clipboardError(err error, other copy.Key) copy.Key {
	if clipboardos.FileAccessDenied(err) {
		return copy.ClipboardAccessRequired
	}
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
	payload, err := clipboardos.Read(a.state.Preferences.Clipboard.Limits.ItemBytes, a.state.Preferences.Clipboard.FilesEnabled)
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
	filesGeneration := cc.filesGeneration.Load()
	a.clipboardWork(func() ([]domain.Clip, error) {
		if payload.Kind == domain.ClipFiles && (cc.filesGeneration.Load() != filesGeneration || !a.preferences().Clipboard.FilesEnabled) {
			return nil, transfer.ErrFormat
		}
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
// become ordinary records and follow the configured count/storage limits.
func refreshClipboard(service *application.Clipboard, limits domain.ClipLimits, includeFiles bool) ([]domain.Clip, error) {
	return service.Update(func(items []domain.Clip) ([]domain.Clip, error) {
		for i, item := range items {
			if includeFiles && item.Pinned && errors.Is(clipboardos.ValidateFiles(item.Payload), transfer.ErrFormat) {
				items[i].Pinned = false
			}
		}
		return domain.PruneClips(items, limits)
	})
}
