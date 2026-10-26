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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/asr"
	"arion.local/desktop/infrastructure/audio"
	"arion.local/desktop/infrastructure/browser"
	"arion.local/desktop/infrastructure/cloudpolish"
	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/infrastructure/keystore"
	"arion.local/desktop/infrastructure/modelruntime"
	"arion.local/desktop/infrastructure/netproxy"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/desktop/infrastructure/rewriting"
	"arion.local/desktop/infrastructure/secrets"
	"arion.local/desktop/infrastructure/settings"
	"arion.local/desktop/infrastructure/usage"
	"arion.local/desktop/infrastructure/vocabulary"
	"arion.local/desktop/infrastructure/wordseg"
	"arion.local/desktop/internal/account"
	"arion.local/desktop/internal/center"
	productnotifications "arion.local/desktop/internal/productnotifications"
	uiapp "arion.local/desktop/ui/app"
	"arion.local/desktop/ui/shared/copy"
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
	// Account supplies transport and timeout test hooks. Provider identity
	// always comes from the compiled center profile and does not contact it.
	Account account.Config
}

// App owns the running desktop application.
type App struct {
	center              center.Profile
	repairedPreferences int
	diagnostics         *diagnostics.Recorder
	exports             *diagnosticExport
	version             string
	opts                Options
	platform            Platform
	prefStore           settings.Store
	catalog             models.Catalog
	store               *provision.Store
	manager             *modelruntime.Manager
	rewriter            *rewriting.Service
	cloud               *cloudpolish.Service
	// history keeps recent dictations per place for rewrite context, in
	// memory only.
	history   *application.History
	mic       *audio.Microphone
	targets   Targets
	ime       InputMethod
	core      *application.Coordinator
	vocabPath string
	// usage persists per-day counts; meter folds session events into them.
	usage usage.Store
	meter application.UsageMeter

	// Main-thread state.
	state   arion.State
	ui      arion.Store
	tray    *mygo.Tray
	overlay *mygo.Window
	panel   *mygo.Window
	// sceneSeq drops a scene peek answered after a newer panel opened.
	sceneSeq                 int
	settingsWin              *mygo.Window
	onboarding               *mygo.Window
	guide                    *guideSession
	trayHint                 *mygo.Window
	views                    map[*mygo.Window]uiapp.View
	testingHold              bool
	escapeActive             bool
	hold                     application.HoldGesture
	harness                  *harness
	monitor                  ports.AudioStream
	vocab                    *vocabulary.Store
	startCue                 *mygoaudio.Sound
	stopCue                  *mygoaudio.Sound
	recommended              string
	updates                  *updates
	updatePolicy             *updatePolicy
	pendingUpdateAction      [2]string
	productNotifications     *productnotifications.Service
	pendingProductActions    [][2]string
	productPermissionPending bool
	resourceJobs             atomic.Int64

	// holdGeneration rejects queued events from a retired native listener.
	holdGeneration uint64

	// Snapshot read by worker goroutines (correction policy, delivery).
	snapshot sync.Mutex
	prefs    application.Preferences
	replaces []domain.Replacement
	// vocabulary indexes replaces once per change; memory is vocab for
	// worker goroutines (edit learning, recall counts).
	vocabulary *domain.Vocabulary
	memory     *vocabulary.Store
	edits      *application.EditTracker
	queueMu    sync.Mutex
	queue      []func()
	queueReady chan struct{}

	accounts           *account.Service
	accountMaintenance accountMaintenance
	accountMu          sync.Mutex
	accountSeq         uint64
	accountCancel      context.CancelFunc
	imeSetupCancel     context.CancelFunc
	imeSetupGeneration uint64
}

// New builds the application without starting the event loop.
func New(opts Options) (*App, error) {
	profile, err := center.Load()
	if err != nil {
		return nil, err
	}
	opts.Account = centerAccountConfig(profile, opts.Account)
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
	a := &App{center: profile, opts: opts, platform: newPlatform(), catalog: catalog, views: map[*mygo.Window]uiapp.View{}, queueReady: make(chan struct{}, 1)}
	a.prefStore = settings.Store{Path: filepath.Join(opts.DataDir, "preferences.json")}
	prefs, repaired, err := a.prefStore.Load()
	if err != nil {
		return nil, err
	}
	if len(repaired) > 0 {
		a.repairedPreferences = len(repaired)
	}
	a.prefs = prefs
	a.store = provision.NewStore(filepath.Join(opts.DataDir, "models"))
	a.store.Offline = opts.Offline
	a.manager = modelruntime.NewManager(catalog, a.store, opts.Threads)
	resources, _ := mygo.App.Path(mygo.PathResources)
	asr.Resources = resources
	a.rewriter = rewriting.New(catalog, a.store, rewriting.Locate(resources), opts.Threads, modelruntime.Platform())
	a.history = application.NewHistory(30 * time.Minute)
	a.mic = audio.NewMicrophone(a.platform)
	a.mic.Select(prefs.Microphone)
	a.mic.SetGain(prefs.MicrophoneGain)
	a.cloud = cloudpolish.New(keystore.OpenItem(keystore.CloudPolishAccount), browser.Open,
		func() []byte { return callbackPage(copy.Locale(a.preferences().UILanguage)) })
	a.cloud.SetModel(prefs.CloudModel)
	// Both engines read the same consent-gated context sources.
	a.targets = a.platform.NewTargets(func() application.Delivery { return a.preferences().Delivery }, TargetGates{
		Site:   func() bool { p := a.preferences(); return p.Polish && p.MatchApp },
		Cursor: func() bool { p := a.preferences(); return p.Polish && p.ContextCursor },
		Screen: func() bool { p := a.preferences(); return p.Polish && p.ContextScreen },
	})
	a.vocabPath = filepath.Join(opts.DataDir, "vocabulary.bin")
	a.usage = usage.Store{Path: filepath.Join(opts.DataDir, "usage.json")}
	capture := opts.Audio
	if capture == nil {
		capture = a.mic
	}
	// The input method ends a system_ime session when it loses the client
	// or the user presses Escape (IME-A03).
	a.ime = a.platform.NewInputMethod(func() {
		a.core.CancelMode(domain.SystemIMEMode)
	}, func() {
		// The input menu's Voice Input item is the tray toggle (IME-06).
		a.post(actions{a}.ToggleDictation)
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
	a.edits = &application.EditTracker{Report: a.learnEdit, Segmenter: wordseg.Segmenter{}}
	deps := application.Dependencies{SelectMode: a.dictationMode, Diagnostics: a, Audio: capture, Recognizers: a.manager, Finalizer: finalizer, Targets: targets, Input: targets,
		Rewriter: polishRouter{a}, History: a.history, Edits: a.edits, NewID: newID}
	if a.ime != nil {
		deps.IMETargets, deps.IMEInput, deps.Composition = a.ime, a.ime, a.ime
	}
	a.core, err = application.NewCoordinator(deps, application.Options{AutoCommit: true, Correction: true, Policy: a.policy,
		Rewrite: a.rewritePolicy, Learning: a.learning, Recalled: a.recalled})
	if err != nil {
		return nil, err
	}
	a.state = arion.State{Preferences: prefs, Platform: modelruntime.Platform(),
		Model:   arion.Model{State: arion.ModelUnselected},
		Account: arion.Account{Phase: arion.AccountSignedOut, Issuer: profile.Issuer, Portal: accountPortal(profile), Disabled: profile.AuthMode == "device", WaitlistDisabled: !profile.Standard()}}
	// Construction stays idle; restoration starts off the main thread after readiness.
	accounts, err := account.New(opts.Account, account.Dependencies{Store: keystore.OpenForProfile(profile.Fingerprint()), Browser: browser.Open,
		CallbackPage: func() []byte { return callbackPage(copy.Locale(a.preferences().UILanguage)) }})
	if err != nil {
		return nil, err
	}
	a.accounts = accounts
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
		Replacements: a.replaces, Vocabulary: a.vocabulary}
}

// rewritePolicy is read when a session starts and preserves the user's
// choice while the worker loads or is unavailable. Approved
// vocabulary reaches the model as data, never as instructions, and only
// the entries relevant to the text (ADR 014).
func (a *App) rewritePolicy() application.RewritePolicy {
	a.snapshot.Lock()
	defer a.snapshot.Unlock()
	if !a.prefs.Polish {
		return application.RewritePolicy{}
	}
	policy := application.RewritePolicy{Enabled: true, Scene: string(domain.SceneChat), FollowApp: a.prefs.MatchApp,
		SceneOverrides: a.prefs.SceneOverrides, Vocabulary: a.vocabulary}
	policy.Context = application.ContextPolicy{Cursor: a.prefs.ContextCursor, Screen: a.prefs.ContextScreen, History: a.prefs.ContextHistory}
	if a.prefs.PolishEngine == application.PolishCloud {
		// Cloud execution runs the shared stages after stop (ADR 023).
		policy.FinalOnly, policy.Timeout = true, cloudRewriteTimeout
	}
	return policy
}

// Run starts the event loop; it returns when the app quits.
func (a *App) Run() error {
	defer a.closeDiagnostics()
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
	mygo.App.OnNotificationAction(a.notificationAction)
	mygo.App.OnQuit(a.closeDiagnostics)
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { actions{a}.CancelInputMethodSetup() })
	mygo.App.OnSecondInstance(func(args []string, _ string) {
		if !hasAppURL(args) {
			a.openSettings()
		}
	})
	// The sign-in callback page links here (Windows forwards it as a second
	// instance's argument, which reaches OnOpenURL too).
	mygo.App.OnOpenURL(func(url string) {
		if isAppURL(url) {
			a.revealWindow()
		}
	})
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
		for role, window := range map[string]*mygo.Window{"overlay": a.overlay, "panel": a.panel, "settings": a.settingsWin, "onboarding": a.onboarding, "update-reminder": func() *mygo.Window {
			if a.updatePolicy != nil {
				return a.updatePolicy.reminder
			}
			return nil
		}()} {
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
	a.version = a.state.Version
	recorder, err := diagnostics.Open(filepath.Join(a.opts.DataDir, "logs"))
	if err != nil {
		a.state.Diagnostics.Phase = arion.DiagnosticsFailed
		log.Print("diagnostics open failed code=IO_FAILED")
	} else {
		a.diagnostics = recorder
		a.record("app_started", nil)
		if a.repairedPreferences > 0 {
			a.recordEntry(diagnostics.Entry{Event: "preferences_repaired", Count: a.repairedPreferences})
		}
	}
	a.setupDiagnosticExport()
	configureInputMethodDiagnostics(a.recordEntry)
	a.startCue, a.stopCue = cue(true), cue(false)
	go a.drain()
	a.core.Subscribe(func(e contracts.Event) {
		if a.diagnostics != nil {
			a.diagnostics.Session(e)
		}
	})
	a.core.Subscribe(func(e contracts.Event) { a.post(func() { a.apply(e) }) })
	if a.harness != nil {
		a.core.Subscribe(a.harness.observe)
		if err := a.harness.serve(a.opts.E2EControl); err != nil {
			a.record("e2e_listen", err)
			mygo.App.Quit()
			return
		}
	}
	a.manager.OnStatus(func(s modelruntime.Status) { a.post(func() { a.applyModel(s) }) })
	a.rewriter.OnStatus(func(s rewriting.Status) { a.post(func() { a.applyPolish(s) }) })
	a.applyPolish(a.rewriter.Status())
	a.runResource(a.startPolish)
	a.cloud.OnStatus(func(s cloudpolish.Status) { a.post(func() { a.applyCloud(s) }) })
	go a.cloud.Load()
	mygo.App.OnQuit(a.rewriter.Disable)
	mygo.App.OnQuit(func() { a.core.Cancel(); a.manager.Close() })
	a.refreshPermissions()
	a.state.InputMethod = arion.InputMethodUnsupported
	if a.ime != nil {
		if err := a.ime.Listen(); err != nil {
			a.record("ime_listen", err)
		} else {
			mygo.App.OnQuit(a.ime.Close)
		}
		a.state.InputMethod = a.ime.State()
		if err := a.ime.ApplyKeyboard(a.state.Preferences.Keyboard, a.state.Preferences.HoldKey); err != nil {
			a.record("keyboard_settings", err)
		}
		go func() {
			update := a.ime.UpdateAvailable()
			a.post(func() { a.state.InputMethodUpdate = update; a.updateInputMethodAutomatically() })
		}()
	}
	a.loadVocabulary(false)
	if days, err := a.usage.Load(); err != nil {
		a.record("usage_load", err)
	} else {
		a.state.Usage = days
	}
	// Offline and scripted profiles never reach the release feed.
	var source updateSource = mygoUpdates{}
	if a.opts.Offline || a.harness != nil || !a.center.Standard() {
		source = nil
	}
	a.updates = newUpdates(source, &a.state.Update, a.post)
	a.updates.setPrereleases(a.state.Preferences.AllowPrereleaseUpdates)
	a.scheduleUpdates()
	if a.pendingUpdateAction[0] != "" {
		a.notificationAction(a.pendingUpdateAction[0], a.pendingUpdateAction[1])
		a.pendingUpdateAction = [2]string{}
	}
	a.startProductNotifications()
	a.startAccountMaintenance()
	a.startHoldKey()
	mygo.App.OnQuit(a.platform.StopHoldKey)
	a.createTray()
	a.createOverlay()
	a.state.Model.Bundles = a.bundles()
	a.runResource(func() { a.activate(a.state.Preferences.ModelOverride) })
	go a.pollPermissions()
	go a.refreshDevicesAsync()
	mygo.Power.OnSuspend(func() { a.record("power_suspend", nil); a.core.Cancel() })
	mygo.Power.OnLockScreen(func() { a.record("screen_locked", nil); a.core.Cancel() })
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
	if u, ok := a.meter.Observe(e); ok {
		a.state.Usage = a.state.Usage.Add(time.Now(), u)
		if err := a.usage.Save(a.state.Usage); err != nil {
			a.record("usage_save", err)
		}
	}
	d := &a.state.Dictation
	if e.SessionID != d.SessionID {
		if e.Kind != contracts.SessionStateChanged || e.State != domain.Preparing {
			return
		}
		*d = arion.Dictation{SessionID: e.SessionID}
		a.dismissLearned()
		a.closeUpdateReminder()
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
	case contracts.RewriteStarted:
		d.Rewriting = true
	case contracts.CorrectionReady:
		d.Final = e.Text
		d.Original = dictationOriginal(e)
		setState(domain.Reviewing)
	case contracts.CommitResultReady:
		d.Outcome, d.Failure = e.Outcome, e.Error
		d.Pasted = a.core.Snapshot().Mode == domain.ShortcutMode && (runtime.GOOS == "darwin" || a.state.Preferences.Delivery == application.DeliveryClipboard) && e.Outcome != contracts.NotApplied
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
		if err := mygo.GlobalShortcut.Register("Escape", func() { a.record("dictation_cancel", nil); a.core.Cancel() }); err == nil {
			a.escapeActive = true
		}
		return
	}
	mygo.GlobalShortcut.Unregister("Escape")
	a.escapeActive = false
}

func (a *App) refreshPermissions() bool {
	next := arion.Permissions{Microphone: a.platform.Microphone(), Accessibility: a.platform.Accessibility(),
		InputMonitoring: a.platform.InputMonitoring(), ScreenRecording: a.platform.ScreenRecording()}
	previous := a.state.Permissions
	changed := next != previous
	if changed && next.Microphone == arion.ConsentGranted && previous.Microphone != arion.ConsentGranted {
		go a.refreshDevicesAsync()
	}
	if changed {
		for capability, value := range map[string]arion.Consent{"microphone": next.Microphone, "accessibility": next.Accessibility, "input_monitoring": next.InputMonitoring, "screen_recording": next.ScreenRecording} {
			a.recordEntry(diagnostics.Entry{Event: "permissions_changed", Capability: capability, Outcome: string(value)})
		}
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
			a.refreshInputMethod()
		})
	}
}

func (a *App) refreshInputMethod() {
	if a.ime != nil && !a.state.InputMethodInstalling && !a.state.InputMethodRemoving {
		a.setInputMethodState(a.ime.State())
		a.updateInputMethodAutomatically()
	}
}

func (a *App) setInputMethodState(state arion.InputMethodState) {
	if a.state.InputMethod == state {
		return
	}
	a.state.InputMethod = state
	a.recordEntry(diagnostics.Entry{Event: "ime_source_state", Outcome: string(state)})
}

func (a *App) refreshDevicesAsync() {
	devices, err := audio.Devices()
	a.record("devices_list", err)
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
		artifact, ok := a.catalog.Artifact(b.Streaming)
		supported := false
		if ok {
			for _, target := range artifact.Platforms {
				if target == platform {
					supported = true
				}
			}
		}
		if !supported {
			continue
		}

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
		a.record("model_activate", err)
	}
}

func (a *App) applyModel(s modelruntime.Status) {
	m := &a.state.Model
	if string(m.State) != string(s.State) || string(m.Phase) != string(s.Phase) || m.Failure != s.Failure {
		a.recordEntry(diagnostics.Entry{Event: "model_status", ModelID: s.Bundle, Outcome: string(s.State), Phase: string(s.Phase), Code: diagnostics.FailureCode(s.Failure)})
	}
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
// user has local polishing on; it never downloads without that choice.
func (a *App) startPolish() {
	a.rewriter.Refresh()
	if p := a.preferences(); p.Polish && p.PolishEngine == application.PolishLocal {
		_ = a.rewriter.Enable(context.Background())
	}
}

func (a *App) applyPolish(s rewriting.Status) {
	p := &a.state.Polish
	if string(p.State) != string(s.State) || string(p.Phase) != string(s.Phase) || p.Failure != s.Failure {
		a.recordEntry(diagnostics.Entry{Event: "rewrite_status", ModelID: rewriting.DefaultModel, Outcome: string(s.State), Phase: string(s.Phase), Code: diagnostics.FailureCode(s.Failure)})
	}
	p.Phase, p.Done, p.Total, p.Failure = arion.ProvisionPhase(s.Phase), s.Done, s.Total, s.Failure
	p.Installed, p.SizeBytes = s.Installed, s.SizeBytes
	p.State = map[rewriting.State]arion.PolishState{rewriting.Unsupported: arion.PolishUnsupported, rewriting.Off: arion.PolishOff,
		rewriting.Provisioning: arion.PolishProvisioning, rewriting.Loading: arion.PolishLoading, rewriting.Ready: arion.PolishReady,
		rewriting.Failed: arion.PolishFailed}[s.State]
}

func (a *App) loadVocabulary(create bool) {
	if a.vocab == nil {
		if _, err := os.Stat(a.vocabPath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				a.record("vocabulary_load", err)
			}
			if !create {
				return
			}
		}
		key, err := secrets.Key("Arion", "vocabulary-key")
		if err != nil {
			a.record("vocabulary_key", err)
			return
		}
		store, err := vocabulary.Open(a.vocabPath, key)
		if err != nil {
			a.record("vocabulary_load", err)
			return
		}
		a.vocab = store
	}
	entries, err := a.vocab.List()
	if err != nil {
		a.record("vocabulary_load", err)
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
	index := domain.NewVocabulary(replaces)
	a.snapshot.Lock()
	a.replaces, a.vocabulary, a.memory = replaces, index, a.vocab
	a.snapshot.Unlock()
}

func (a *App) savePreferences(prefs application.Preferences) bool {
	if err := a.prefStore.Save(prefs); err != nil {
		a.record("preferences_save", err)
		return false
	}
	a.snapshot.Lock()
	a.prefs = prefs
	a.snapshot.Unlock()
	a.state.Preferences = prefs
	return true
}

func (a *App) startMonitor() {
	if a.updateApplying() {
		return
	}
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
		a.resourceJobs.Add(1)
		go func() { defer a.resourceJobs.Add(-1); _ = stream.Stop() }()
	}
	a.state.MonitorLevels = nil
}

// Cleanup may succeed while organization fails. The source remains
// copyable whenever delivered text differs, regardless of that outcome.
func dictationOriginal(e contracts.Event) string {
	if e.Source == e.Text {
		return ""
	}
	return e.Source
}

// centerAccountConfig adapts immutable package configuration to the account service.
func centerAccountConfig(profile center.Profile, cfg account.Config) account.Config {
	cfg.Disabled = profile.AuthMode == "device"
	cfg.DisableWaitlist = !profile.Standard()
	cfg.Entitlement = profile.Entitlement
	cfg.Issuer, cfg.ClientID, cfg.Resource = profile.Issuer, profile.ClientID, profile.Resource
	cfg.Scopes = append([]string(nil), profile.Scopes...)
	return cfg
}

// accountPortal is the AstraFoundry account portal of the standard profile.
// A custom issuer's web pages are unknown, so it has none.
func accountPortal(profile center.Profile) string {
	if !profile.Standard() || profile.AuthMode == "device" {
		return ""
	}
	return strings.TrimSuffix(profile.Issuer, "/") + "/account"
}
