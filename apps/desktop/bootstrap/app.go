// Package bootstrap is the desktop composition root: it wires the core
// coordinator to OS adapters, the model runtime and the native UI windows.
package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/audio"
	"arion.local/desktop/infrastructure/modelruntime"
	"arion.local/desktop/infrastructure/netproxy"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/desktop/infrastructure/rewriting"
	"arion.local/desktop/infrastructure/secrets"
	"arion.local/desktop/infrastructure/settings"
	"arion.local/desktop/infrastructure/vocabulary"
	uiapp "arion.local/desktop/ui/app"
	arion "arion.local/desktop/ui/shared/runtime"
	"arion.local/models"
	"github.com/egoist/mygo"
	mygoaudio "github.com/egoist/mygo/audio"
)

// Options configure the composition root. Zero values select the product
// defaults; overrides exist for isolated E2E profiles.
type Options struct {
	// Audio replaces the microphone with a fixture source for automated E2E
	// scenarios. The product never sets it.
	Audio ports.AudioCapture
	// DataDir holds preferences, the model store and vocabulary.
	DataDir string
	// Threads bounds inference threads.
	Threads int
	// SkipOnboarding starts without the first-run window (E2E).
	SkipOnboarding bool
	// Offline refuses model downloads, for offline scenarios.
	Offline bool
	// E2EControl, when set, is the path of a local control socket for owned
	// product scenarios; it swaps the microphone for scripted fixture audio.
	E2EControl string
}

// App owns the running desktop application.
type App struct {
	opts      Options
	platform  Platform
	prefStore settings.Store
	catalog   models.Catalog
	store     *provision.Store
	manager   *modelruntime.Manager
	rewriter  *rewriting.Service
	mic       *audio.Microphone
	targets   Targets
	ime       InputMethod
	core      *application.Coordinator
	vocabPath string

	// Main-thread state.
	state           arion.State
	ui              arion.Store
	tray            *mygo.Tray
	overlay         *mygo.Window
	panel           *mygo.Window
	settingsWin     *mygo.Window
	onboarding      *mygo.Window
	guide           *guideSession
	views           map[*mygo.Window]uiapp.View
	testingShortcut bool
	shortcutActive  bool
	escapeActive    bool
	hold            application.HoldGesture
	harness         *harness
	monitor         ports.AudioStream
	vocab           *vocabulary.Store
	startCue        *mygoaudio.Sound
	stopCue         *mygoaudio.Sound
	recommended     string
	updates         *updates

	// Snapshot read by worker goroutines (correction policy, delivery).
	snapshot   sync.Mutex
	prefs      application.Preferences
	replaces   []domain.Replacement
	queueMu    sync.Mutex
	queue      []func()
	queueReady chan struct{}
}

// New builds the application without starting the event loop.
func New(opts Options) (*App, error) {
	if opts.DataDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		opts.DataDir = filepath.Join(base, "Arion")
	}
	// Release checks use the default transport; like model downloads they
	// follow the system proxy when the app was not started from a shell.
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport.Proxy = netproxy.Proxy
	}
	if opts.Threads <= 0 {
		opts.Threads = min(4, max(2, runtime.NumCPU()/2))
	}
	catalog, err := models.Load()
	if err != nil {
		return nil, err
	}
	a := &App{opts: opts, platform: newPlatform(), catalog: catalog, views: map[*mygo.Window]uiapp.View{}, queueReady: make(chan struct{}, 1)}
	a.prefStore = settings.Store{Path: filepath.Join(opts.DataDir, "preferences.json")}
	prefs, repaired, err := a.prefStore.Load()
	if err != nil {
		return nil, err
	}
	if len(repaired) > 0 {
		log.Printf("preferences repaired fields=%v", repaired)
	}
	a.prefs = prefs
	a.store = provision.NewStore(filepath.Join(opts.DataDir, "models"))
	a.store.Offline = opts.Offline
	a.manager = modelruntime.NewManager(catalog, a.store, opts.Threads)
	resources, _ := mygo.App.Path(mygo.PathResources)
	a.rewriter = rewriting.New(catalog, a.store, rewriting.Locate(resources), opts.Threads, modelruntime.Platform())
	a.mic = audio.NewMicrophone(a.platform)
	a.mic.Select(prefs.Microphone)
	a.mic.SetGain(prefs.MicrophoneGain)
	a.targets = a.platform.NewTargets(func() application.Delivery { return a.preferences().Delivery })
	a.vocabPath = filepath.Join(opts.DataDir, "vocabulary.bin")
	capture := opts.Audio
	if capture == nil {
		capture = a.mic
	}
	// The input method ends a system_ime session when it loses the client
	// or the user presses Escape (IME-A03).
	a.ime = a.platform.NewInputMethod(func() {
		if a.preferences().Mode == domain.SystemIMEMode {
			a.core.Cancel()
		}
	})
	var finalizer ports.Finalizer = a.manager
	var targets Targets = a.targets
	if opts.E2EControl != "" {
		a.harness = newHarness()
		a.harness.app = a
		capture = a.harness.audio
		finalizer = gatedFinalizer{h: a.harness, inner: a.manager}
		targets = countingTargets{Targets: a.targets, h: a.harness}
	}
	deps := application.Dependencies{Audio: capture, Recognizers: a.manager, Finalizer: finalizer, Targets: targets, Input: targets,
		Rewriter: a.rewriter, NewID: newID}
	if a.ime != nil {
		deps.IMETargets, deps.IMEInput, deps.Composition = a.ime, a.ime, a.ime
	}
	a.core, err = application.NewCoordinator(deps, application.Options{Mode: prefs.Mode, AutoCommit: true, Correction: true, Policy: a.policy,
		Rewrite: a.rewritePolicy})
	if err != nil {
		return nil, err
	}
	a.state = arion.State{Preferences: prefs, Platform: modelruntime.Platform(),
		Shortcut: arion.Shortcut{Accelerator: prefs.Shortcut}, Model: arion.Model{State: arion.ModelUnselected}}
	a.ui = arion.Store{State: &a.state, Actions: actions{a}}
	return a, nil
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (a *App) preferences() application.Preferences {
	a.snapshot.Lock()
	defer a.snapshot.Unlock()
	return a.prefs
}

// policy is read by the session goroutine when a dictation finalizes.
func (a *App) policy() domain.CorrectionPolicy {
	a.snapshot.Lock()
	defer a.snapshot.Unlock()
	return domain.CorrectionPolicy{LatinCase: a.manager.LatinCase(), RemoveFillers: a.prefs.RemoveFillers,
		Replacements: append([]domain.Replacement(nil), a.replaces...)}
}

// rewritePolicy is read when a session starts: the live pipeline runs only
// when the user turned it on and the worker is serving. Approved
// vocabulary reaches the model as data, never as instructions.
func (a *App) rewritePolicy() application.RewritePolicy {
	a.snapshot.Lock()
	defer a.snapshot.Unlock()
	if !a.prefs.Polish || !a.rewriter.Ready() {
		return application.RewritePolicy{}
	}
	terms := make([]string, 0, len(a.replaces))
	for _, r := range a.replaces {
		terms = append(terms, r.From+" → "+r.To)
	}
	return application.RewritePolicy{Enabled: true, Scene: "chat", Terms: terms}
}

// Run starts the event loop; it returns when the app quits.
func (a *App) Run() error {
	if !mygo.App.RequestSingleInstanceLock() {
		return errors.New("Arion is already running")
	}
	a.Start()
	return mygo.App.Run()
}

// Start registers the app's lifecycle with MyGo without running the event
// loop, for hosts such as native E2E tests that own the loop.
func (a *App) Start() {
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnSecondInstance(func([]string, string) { a.openSettings() })
	mygo.App.WhenReady(a.ready)
}

// OpenSettings shows the settings window.
func (a *App) OpenSettings() { mygo.RunOnMain(a.openSettings) }

// OpenOnboarding shows the first-run window.
func (a *App) OpenOnboarding() { mygo.RunOnMain(a.openOnboarding) }

// Windows lists the open native windows by role, for E2E capture.
func (a *App) Windows() map[string]*mygo.Window {
	windows := map[string]*mygo.Window{}
	mygo.RunOnMain(func() {
		for role, window := range map[string]*mygo.Window{"overlay": a.overlay, "panel": a.panel, "settings": a.settingsWin, "onboarding": a.onboarding} {
			if window != nil {
				windows[role] = window
			}
		}
	})
	return windows
}

// Update runs fn on the main thread with the UI state and redraws.
func (a *App) Update(fn func(*arion.State)) {
	mygo.RunOnMain(func() {
		fn(&a.state)
		a.invalidate()
	})
}

// Store exposes the UI store for E2E drivers.
func (a *App) Store() arion.Store { return a.ui }

// Coordinator exposes the session coordinator for E2E drivers.
func (a *App) Coordinator() *application.Coordinator { return a.core }

func (a *App) ready() {
	a.state.Version = mygo.App.Version()
	a.startCue, a.stopCue = cue(true), cue(false)
	go a.drain()
	a.core.Subscribe(func(e contracts.Event) { a.post(func() { a.apply(e) }) })
	if a.harness != nil {
		a.core.Subscribe(a.harness.observe)
		if err := a.harness.serve(a.opts.E2EControl); err != nil {
			log.Printf("e2e control unavailable: %v", err)
			mygo.App.Quit()
			return
		}
	}
	a.manager.OnStatus(func(s modelruntime.Status) { a.post(func() { a.applyModel(s) }) })
	a.rewriter.OnStatus(func(s rewriting.Status) { a.post(func() { a.applyPolish(s) }) })
	a.applyPolish(a.rewriter.Status())
	go a.startPolish()
	mygo.App.OnQuit(a.rewriter.Disable)
	a.refreshPermissions()
	a.state.InputMethod = arion.InputMethodUnsupported
	if a.ime != nil {
		if err := a.ime.Listen(); err != nil {
			log.Printf("input method socket unavailable: %v", err)
		}
		a.state.InputMethod = a.ime.State()
	}
	a.loadVocabulary(false)
	// Offline and scripted profiles never reach the release feed.
	var source updateSource = mygoUpdates{}
	if a.opts.Offline || a.harness != nil {
		source = nil
	}
	a.updates = newUpdates(source, &a.state.Update, a.post)
	a.scheduleUpdates()
	a.registerShortcut(a.state.Preferences.Shortcut)
	a.startHoldKey()
	mygo.App.OnQuit(a.platform.StopHoldKey)
	a.createTray()
	a.createOverlay()
	a.state.Model.Bundles = a.bundles()
	go a.activate(a.state.Preferences.ModelOverride)
	go a.pollPermissions()
	go a.refreshDevicesAsync()
	mygo.Power.OnSuspend(a.core.Cancel)
	mygo.Power.OnLockScreen(a.core.Cancel)
	if !a.state.Preferences.OnboardingCompleted && !a.opts.SkipOnboarding {
		a.openOnboarding()
	}
}

// post queues fn for the main thread without blocking the caller, so a
// coordinator event can never wait on a main thread that waits on it.
func (a *App) post(fn func()) {
	a.queueMu.Lock()
	a.queue = append(a.queue, fn)
	a.queueMu.Unlock()
	select {
	case a.queueReady <- struct{}{}:
	default:
	}
}

func (a *App) drain() {
	for range a.queueReady {
		a.queueMu.Lock()
		batch := a.queue
		a.queue = nil
		a.queueMu.Unlock()
		if len(batch) == 0 {
			continue
		}
		mygo.RunOnMain(func() {
			for _, fn := range batch {
				fn()
			}
			a.invalidate()
		})
	}
}

func (a *App) invalidate() {
	for window := range a.views {
		window.Invalidate()
	}
}

const levelHistory = 48

func appendLevel(levels []float32, level float32) []float32 {
	levels = append(levels, level)
	if len(levels) > levelHistory {
		levels = levels[len(levels)-levelHistory:]
	}
	return levels
}

// apply folds a coordinator event into the UI state on the main thread.
func (a *App) apply(e contracts.Event) {
	d := &a.state.Dictation
	if e.SessionID != d.SessionID {
		if e.Kind != contracts.SessionStateChanged || e.State != domain.Preparing {
			return
		}
		*d = arion.Dictation{SessionID: e.SessionID}
		a.stopMonitor()
		a.positionOverlay()
	}
	now := time.Now()
	setState := func(state domain.SessionState) {
		if d.State != state {
			d.State, d.Changed = state, now
		}
	}
	switch e.Kind {
	case contracts.SessionStateChanged:
		setState(e.State)
		switch e.State {
		case domain.Capturing:
			a.play(a.startCue)
		case domain.Finalizing:
			a.play(a.stopCue)
		}
	case contracts.AudioLevel:
		d.Levels = appendLevel(d.Levels, e.Level)
	case contracts.DraftReplaced:
		d.Draft = e.Text
	case contracts.CorrectionReady:
		d.Final = e.Text
		d.Original = ""
		if e.Rewrite == string(domain.RewriteAccepted) && e.Source != e.Text {
			d.Original = e.Source
		}
		setState(domain.Reviewing)
	case contracts.CommitResultReady:
		d.Outcome, d.Failure = e.Outcome, e.Error
		d.Pasted = a.state.Preferences.Delivery == application.DeliveryClipboard && e.Outcome != contracts.NotApplied
		a.state.LastText, a.state.LastOriginal = e.Text, d.Original
		setState(e.State)
	case contracts.ErrorOccurred:
		if e.Error == domain.CodeSessionBusy {
			return
		}
		d.Failure, d.FailureDetail = e.Error, e.Detail
		if e.Text != "" {
			a.state.LastText, a.state.LastOriginal = e.Text, ""
		}
		setState(e.State)
	}
	a.updateEscape()
	a.updateTray()
}

func (a *App) play(sound *mygoaudio.Sound) {
	if a.state.Preferences.Sounds && sound != nil {
		sound.Play()
	}
}

// updateEscape holds Escape globally only while a session can be cancelled.
func (a *App) updateEscape() {
	active := false
	switch a.state.Dictation.State {
	case domain.Preparing, domain.Capturing, domain.Finalizing:
		active = true
	}
	if active == a.escapeActive {
		return
	}
	if active {
		if err := mygo.GlobalShortcut.Register("Escape", a.core.Cancel); err == nil {
			a.escapeActive = true
		}
		return
	}
	mygo.GlobalShortcut.Unregister("Escape")
	a.escapeActive = false
}

func (a *App) registerShortcut(accelerator string) error {
	if a.shortcutActive {
		mygo.GlobalShortcut.Unregister(a.state.Shortcut.Accelerator)
		a.shortcutActive = false
	}
	err := mygo.GlobalShortcut.Register(accelerator, a.onShortcut)
	if err == nil {
		a.shortcutActive = true
		a.state.Shortcut.Accelerator = accelerator
	}
	a.state.Shortcut.Registered = a.shortcutActive
	return err
}

func (a *App) onShortcut() {
	a.state.Shortcut.Pressed = time.Now()
	if a.testingShortcut {
		a.invalidate()
		return
	}
	a.core.Toggle()
}

func (a *App) refreshPermissions() bool {
	next := arion.Permissions{Microphone: a.platform.Microphone(), Accessibility: a.platform.Accessibility(),
		InputMonitoring: a.platform.InputMonitoring()}
	previous := a.state.Permissions
	changed := next != previous
	if changed && next.Microphone == arion.ConsentGranted && previous.Microphone != arion.ConsentGranted {
		go a.refreshDevicesAsync()
	}
	a.state.Permissions = next
	a.refreshHoldConsent(previous.InputMonitoring)
	return changed
}

// pollPermissions notices consent given in System Settings, which sends no
// notification.
func (a *App) pollPermissions() {
	for range time.Tick(time.Second) {
		a.post(func() {
			if a.refreshPermissions() {
				a.updateTray()
			}
			if a.ime != nil {
				a.state.InputMethod = a.ime.State()
			}
		})
	}
}

func (a *App) refreshDevicesAsync() {
	devices, err := audio.Devices()
	if err != nil {
		return
	}
	a.post(func() {
		a.state.Devices = a.state.Devices[:0]
		for _, device := range devices {
			a.state.Devices = append(a.state.Devices, arion.Device{ID: device.ID, Name: device.Name, Default: device.Default})
		}
	})
}

// bundles lists every catalog bundle with its evaluation summary for this
// hardware class, marking the automatic choice.
func (a *App) bundles() []arion.Bundle {
	recommended, selection, err := modelruntime.SelectDefault(a.catalog)
	if err == nil {
		a.recommended = recommended.ID
		a.state.Model.LatencyMeasured = selection.LatencyMeasured
	}
	platform := modelruntime.Platform()
	var out []arion.Bundle
	for _, b := range a.catalog.Bundles {
		bundle := arion.Bundle{ID: b.ID, Streaming: b.Streaming, Finalizer: b.Finalizer, SizeBytes: a.catalog.DownloadBytes(b), Recommended: b.ID == a.recommended}
		for _, e := range b.Evaluations {
			if e.Hardware == platform || !bundle.Evaluated {
				bundle.Evaluated, bundle.ErrorRates, bundle.ReviewP95Ms, bundle.RTF = true, e.ErrorRates, e.StopToReviewP95Ms, e.RealTimeFactor
			}
		}
		out = append(out, bundle)
	}
	return out
}

// activate provisions and loads the override, or the automatic choice.
func (a *App) activate(override string) {
	id := override
	if id == "" {
		id = a.recommended
	}
	bundle, ok := a.catalog.Bundle(id)
	if !ok {
		a.post(func() {
			a.state.Model.State, a.state.Model.Failure = arion.ModelUnselected, domain.CodeEvaluationUnavailable
		})
		return
	}
	a.post(func() {
		a.state.Model.Automatic = override == ""
		for _, b := range a.state.Model.Bundles {
			if b.ID == bundle.ID {
				a.state.Model.Active = b
			}
		}
	})
	if err := a.manager.Activate(contextBackground(), bundle); err != nil {
		log.Printf("model activation failed bundle=%s code=%s", bundle.ID, domain.CodeOf(err))
	}
}

func (a *App) applyModel(s modelruntime.Status) {
	m := &a.state.Model
	m.Done, m.Total, m.Failure = s.Done, s.Total, s.Failure
	m.Phase = arion.ProvisionPhase(s.Phase)
	switch s.State {
	case modelruntime.StateProvisioning:
		m.State = arion.ModelProvisioning
	case modelruntime.StateLoading:
		m.State = arion.ModelLoading
	case modelruntime.StateReady:
		m.State = arion.ModelReady
	case modelruntime.StateFailed:
		m.State = arion.ModelFailed
	case modelruntime.StateUnselected:
		m.State = arion.ModelUnselected
	}
	a.updateTray()
}

// startPolish refreshes installation state and starts the worker when the
// user has the capability on; it never downloads without that choice.
func (a *App) startPolish() {
	a.rewriter.Refresh()
	if a.preferences().Polish {
		_ = a.rewriter.Enable(context.Background())
	}
}

func (a *App) applyPolish(s rewriting.Status) {
	p := &a.state.Polish
	p.Phase, p.Done, p.Total, p.Failure = arion.ProvisionPhase(s.Phase), s.Done, s.Total, s.Failure
	p.Installed, p.SizeBytes = s.Installed, s.SizeBytes
	p.State = map[rewriting.State]arion.PolishState{rewriting.Unsupported: arion.PolishUnsupported, rewriting.Off: arion.PolishOff,
		rewriting.Provisioning: arion.PolishProvisioning, rewriting.Loading: arion.PolishLoading, rewriting.Ready: arion.PolishReady,
		rewriting.Failed: arion.PolishFailed}[s.State]
}

func (a *App) loadVocabulary(create bool) {
	if a.vocab == nil {
		if _, err := os.Stat(a.vocabPath); err != nil && !create {
			return
		}
		key, err := secrets.Key("Arion", "vocabulary-key")
		if err != nil {
			log.Printf("vocabulary key unavailable: %v", err)
			return
		}
		store, err := vocabulary.Open(a.vocabPath, key)
		if err != nil {
			return
		}
		a.vocab = store
	}
	entries, err := a.vocab.List()
	if err != nil {
		log.Printf("vocabulary unreadable: %v", err)
		return
	}
	a.state.Vocabulary = a.state.Vocabulary[:0]
	var replaces []domain.Replacement
	for _, entry := range entries {
		a.state.Vocabulary = append(a.state.Vocabulary, arion.Replacement{ID: entry.ID, From: entry.From, To: entry.To})
		if entry.Approved {
			replaces = append(replaces, domain.Replacement{From: entry.From, To: entry.To})
		}
	}
	a.snapshot.Lock()
	a.replaces = replaces
	a.snapshot.Unlock()
}

func (a *App) savePreferences(prefs application.Preferences) {
	if err := a.prefStore.Save(prefs); err != nil {
		log.Printf("preferences not saved: %v", err)
	}
	a.snapshot.Lock()
	a.prefs = prefs
	a.snapshot.Unlock()
	a.state.Preferences = prefs
}

func (a *App) startMonitor() {
	if a.monitor != nil || a.state.Permissions.Microphone != arion.ConsentGranted {
		return
	}
	switch a.state.Dictation.State {
	case domain.Preparing, domain.Capturing:
		return
	}
	stream, err := a.mic.Start(contextBackground(), contracts.Identity{SessionID: "monitor", Generation: 1}, func(frame contracts.Frame) {
		level := application.Level(frame.Samples)
		a.post(func() { a.state.MonitorLevels = appendLevel(a.state.MonitorLevels, level) })
	})
	if err != nil {
		return
	}
	a.monitor = stream
}

func (a *App) stopMonitor() {
	if a.monitor != nil {
		stream := a.monitor
		a.monitor = nil
		go stream.Stop()
	}
	a.state.MonitorLevels = nil
}
