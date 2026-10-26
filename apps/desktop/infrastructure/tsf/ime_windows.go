//go:build windows

// Package tsf installs and observes the native Windows text service.
package tsf

import (
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/imebridge"
)

type State struct {
	Registered      bool   `json:"registered"`
	Enabled         bool   `json:"enabled"`
	Selected        bool   `json:"selected"`
	Build           string `json:"build"`
	Legacy          bool   `json:"legacy"`
	UpgradeRequired bool   `json:"upgradeRequired"`
	Loader          bool   `json:"loader"`
}

func (s State) current(build string) bool {
	return s.Registered && !s.Legacy && !s.UpgradeRequired && s.Build == build
}

type IME struct {
	*imebridge.IME
	resources      func() (string, error)
	mu             sync.Mutex
	maintenance    sync.Mutex
	observed       State
	initErr        error
	observationErr error
	polling        bool
	cancel         context.CancelFunc
	build          string
	buildErr       error
	validated      bool
}

func New(resources func() (string, error), onLeave, onDictate func()) *IME {
	path, err := imebridge.WindowsPipePath()
	bridge := imebridge.New(path, onLeave, onDictate)
	bridge.RequireBeginAcknowledgement()
	bridge.RequireClientVerification(verifyWindowsClient)
	return &IME{IME: bridge, resources: resources, initErr: err}
}
func (m *IME) Listen() error {
	if m.initErr != nil {
		return m.initErr
	}
	if err := m.IME.Listen(); err != nil {
		return err
	}
	m.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.polling = true
	m.mu.Unlock()
	go func() {
		build, err := m.validate()
		m.mu.Lock()
		m.build, m.buildErr, m.validated = build, err, true
		m.mu.Unlock()
		timer := time.NewTicker(2 * time.Second)
		defer timer.Stop()
		for {
			observation, done := context.WithTimeout(ctx, time.Second)
			_, _ = m.Query(observation)
			done()
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
	}()
	return nil
}
func (m *IME) Close() {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()
	m.IME.Close()
}
func (m *IME) paths() (string, string, error) {
	root, err := m.resources()
	if err != nil {
		return "", "", err
	}
	for _, directory := range []string{filepath.Join(root, "ime"), filepath.Join(root, "windows-amd64", "ime")} {
		helper, dll := filepath.Join(directory, "arion-ime.exe"), filepath.Join(directory, "ArionTSF.dll")
		if info, err := os.Stat(helper); err == nil && !info.IsDir() {
			return helper, dll, nil
		}
	}
	return "", "", errors.New("Windows TSF runtime is not bundled")
}
func (m *IME) command(ctx context.Context, args ...string) ([]byte, error) {
	helper, _, err := m.paths()
	if err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, helper, args...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	data, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Windows TSF %s: %w (%s)", args[0], err, strings.TrimSpace(string(data)))
	}
	return data, nil
}
func (m *IME) Query(ctx context.Context) (State, error) {
	data, err := m.command(ctx, "query")
	if err != nil {
		m.mu.Lock()
		m.observationErr = err
		m.mu.Unlock()
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		m.mu.Lock()
		m.observationErr = err
		m.mu.Unlock()
		return State{}, err
	}
	m.mu.Lock()
	m.observed = state
	m.observationErr = nil
	if m.validated && m.buildErr != nil {
		m.observationErr = m.buildErr
	}
	m.mu.Unlock()
	return state, nil
}

type updateDeferred struct{}

func (updateDeferred) Error() string {
	return "Windows input method update deferred until the source is unused"
}
func (updateDeferred) InputMethodUpdateDeferred() bool { return true }
func (m *IME) Capture(ctx context.Context, id contracts.Identity) (contracts.TargetSnapshot, error) {
	m.mu.Lock()
	err, validated := m.buildErr, m.validated
	m.mu.Unlock()
	if !validated {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "Windows TSF payload validation is pending")
	}
	if err != nil {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeArtifactUnverified, domain.DetailInputMethodArtifactUnverified)
	}
	return m.IME.Capture(ctx, id)
}

// Observed preserves the last known selection when native observation fails;
// transport failure must never silently reroute an owned IME session.
func (m *IME) Observed() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.observed
}
func (m *IME) ObservationError() error { m.mu.Lock(); defer m.mu.Unlock(); return m.observationErr }
func (m *IME) SelectedForSession() bool {
	if m.IME.ActiveClient() {
		return true
	}
	selected, err := selectedNative()
	if err != nil {
		m.mu.Lock()
		m.observationErr = err
		m.mu.Unlock()
		return true
	}
	return selected
}

type payload struct {
	Schema                 int               `json:"schema"`
	Architecture           string            `json:"architecture"`
	Files                  map[string]string `json:"files"`
	KeyboardManifestSHA256 string            `json:"keyboardManifestSHA256"`
}

func (m *IME) validate() (string, error) {
	helper, dll, err := m.paths()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(dll), "runtime.json"))
	if err != nil {
		return "", err
	}
	var metadata payload
	if json.Unmarshal(data, &metadata) != nil || metadata.Schema != 2 || metadata.Architecture != "amd64" || len(metadata.Files) != 3 {
		return "", domain.Fail(domain.CodeArtifactUnverified, "Windows TSF manifest is invalid")
	}
	for _, file := range []string{helper, dll, filepath.Join(filepath.Dir(dll), "ArionTSFLoader.dll")} {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		expected := metadata.Files[filepath.Base(file)]
		if expected == "" || hex.EncodeToString(sum[:]) != expected {
			return "", domain.Fail(domain.CodeArtifactUnverified, "Windows TSF payload hash differs")
		}
		image, err := pe.Open(file)
		if err != nil {
			return "", err
		}
		architecture := image.Machine
		image.Close()
		if architecture != pe.IMAGE_FILE_MACHINE_AMD64 {
			return "", domain.Fail(domain.CodeArtifactUnverified, "Windows TSF payload is not x64")
		}
	}
	if err := validateKeyboardRuntime(filepath.Join(filepath.Dir(filepath.Dir(dll)), "keyboard"), metadata.KeyboardManifestSHA256); err != nil {
		return "", err
	}
	return metadata.Files["ArionTSF.dll"], nil
}
func (m *IME) Install() error {
	m.maintenance.Lock()
	defer m.maintenance.Unlock()
	return m.install(false)
}

func (m *IME) UpdateAutomatically() error {
	m.maintenance.Lock()
	defer m.maintenance.Unlock()
	return m.install(true)
}

func (m *IME) CanUpdateWhileSelected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.observationErr == nil && m.observed.Loader && !m.observed.Legacy && m.observed.Registered && m.observed.Enabled
}

func (m *IME) install(automatic bool) error {
	build, err := m.validate()
	if err != nil {
		return err
	}
	if err = m.EnsureListening(); err != nil {
		return err
	}
	helper, dll, err := m.paths()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	m.BeginUpdate()
	defer m.EndUpdate()
	return maintainRegistration(ctx, build, dll, automatic, m.Query, m.IME.ActiveClient,
		func(ctx context.Context, args []string) error { return elevated(ctx, helper, args) },
		func(ctx context.Context, action string) error { _, err := m.command(ctx, action); return err }, 250*time.Millisecond)
}

// Ordinary-user observation and TIP maintenance surround exactly one approved
// machine transaction. This boundary is tested without executing a helper.
func maintainRegistration(ctx context.Context, build, dll string, automatic bool, query func(context.Context) (State, error), active func() bool, machine func(context.Context, []string) error, user func(context.Context, string) error, interval time.Duration) error {
	state, err := query(ctx)
	if err != nil {
		return err
	}
	if state.current(build) {
		// An explicit retry may complete user enablement after a committed
		// machine upgrade; background maintenance never enables a disabled TIP.
		if automatic || state.Enabled {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := user(ctx, "enable"); err != nil {
			return err
		}
		return observeEnablement(ctx, build, query, interval)
	}
	if automatic && (!state.Registered || !state.Enabled) {
		return updateDeferred{}
	}
	// A stable loader publishes the next core while live editors retain their
	// current compatible core. Direct legacy registrations still need an idle
	// migration; they cannot acquire loader behavior through a registry write.
	if state.Registered && !state.Loader && (state.Selected || active()) {
		return updateDeferred{}
	}
	args := []string{"install", dll, build}
	if state.Registered {
		_, digestError := hex.DecodeString(state.Build)
		if len(state.Build) != 64 || digestError != nil || strings.ToLower(state.Build) != state.Build {
			return domain.Fail(domain.CodeArtifactUnverified, "installed Windows TSF build is invalid")
		}
		args = []string{"update", dll, build, state.Build}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := machine(ctx, args); err != nil {
		var deferred interface{ InputMethodUpdateDeferred() bool }
		if errors.As(err, &deferred) && deferred.InputMethodUpdateDeferred() {
			// Refresh selected state before bootstrap considers another attempt.
			if _, observed := query(ctx); observed != nil {
				return observed
			}
		}
		return err
	}
	observation, done := context.WithTimeout(ctx, 30*time.Second)
	defer done()
	if err := observeRegistration(observation, build, query, interval); err != nil {
		return err
	}
	if state.Loader && !state.Legacy {
		// Publishing a core preserves the existing profile and per-user choice.
		// Do not toggle enablement or interrupt the editor's live composition.
		return nil
	}
	// User TIP changes run as the desktop user, never the elevated account.
	if state.Registered && state.Enabled {
		if state.Legacy {
			if err := user(ctx, "disable-legacy"); err != nil {
				return err
			}
		}
		if err := user(ctx, "enable"); err != nil {
			return err
		}
		return observeEnablement(observation, build, query, interval)
	}
	return nil
}

type setupError struct {
	stage string
	err   error
}

func (e setupError) Error() string                 { return fmt.Sprintf("Windows TSF %s: %v", e.stage, e.err) }
func (e setupError) Unwrap() error                 { return e.err }
func (e setupError) InputMethodSetupStage() string { return e.stage }

// A successful native register operation precedes discovery by independent
// processes. Observe it without repeating registration or hiding API failures.
func observeRegistration(ctx context.Context, build string, query func(context.Context) (State, error), interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return setupError{"discovery", err}
		}
		state, err := query(ctx)
		if err != nil {
			return setupError{"discovery", err}
		}
		if state.Registered {
			if !state.current(build) {
				return domain.Fail(domain.CodeArtifactUnverified, "observed Windows TSF installed build differs")
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return setupError{"discovery", ctx.Err()}
		case <-ticker.C:
		}
	}
}
func (m *IME) Activate(ctx context.Context) error {
	build, err := m.validate()
	if err != nil {
		return err
	}
	if err = m.EnsureListening(); err != nil {
		return setupError{"connection", err}
	}
	state, err := m.Query(ctx)
	if err != nil {
		return setupError{"discovery", err}
	}
	if !state.Registered {
		return setupError{"discovery", errors.New("Windows TSF is not registered")}
	}
	if !state.current(build) {
		return domain.Fail(domain.CodeArtifactUnverified, "installed Windows TSF build differs")
	}
	if _, err = m.command(ctx, "enable"); err != nil {
		return setupError{"enable", err}
	}
	if err = repairDisabledRegistration(ctx, build, m.Query, func() error {
		helper, _, err := m.paths()
		if err != nil {
			return err
		}
		if err = elevated(ctx, helper, []string{"repair"}); err != nil {
			return err
		}
		_, err = m.command(ctx, "enable")
		return err
	}); err != nil {
		return err
	}
	if err = observeEnablement(ctx, build, m.Query, 250*time.Millisecond); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return setupError{"selection", err}
	}
	if _, err = m.command(ctx, "select"); err != nil {
		return setupError{"selection", err}
	}
	timer := time.NewTicker(250 * time.Millisecond)
	defer timer.Stop()
	for {
		state, err = m.Query(ctx)
		if err != nil {
			return setupError{"selection", err}
		}
		if !state.current(build) {
			return domain.Fail(domain.CodeArtifactUnverified, "Windows TSF registration changed during selection")
		}
		if state.Enabled && state.Selected {
			return nil
		}
		select {
		case <-ctx.Done():
			return setupError{"selection", ctx.Err()}
		case <-timer.C:
		}
	}
}

// Older installations disabled the machine profile while reporting the user
// preference enabled. Repair only during an explicit activation, after the
// user installation request, and never replace the installed executable.
func repairDisabledRegistration(ctx context.Context, build string, query func(context.Context) (State, error), repair func() error) error {
	if err := ctx.Err(); err != nil {
		return setupError{"enable", err}
	}
	state, err := query(ctx)
	if err != nil {
		return setupError{"enable", err}
	}
	if !state.current(build) {
		return domain.Fail(domain.CodeArtifactUnverified, "Windows TSF registration changed before repair")
	}
	if state.Enabled {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return setupError{"enable", err}
	}
	if err := repair(); err != nil {
		return setupError{"enable", err}
	}
	return nil
}

// User-list installation and observed enablement are separate stages. Do not
// request selection while Windows still reports the profile disabled.
func observeEnablement(ctx context.Context, build string, query func(context.Context) (State, error), interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return setupError{"enable", err}
		}
		state, err := query(ctx)
		if err != nil {
			return setupError{"enable", err}
		}
		if !state.current(build) {
			return domain.Fail(domain.CodeArtifactUnverified, "Windows TSF registration changed during enablement")
		}
		if state.Enabled {
			return nil
		}
		select {
		case <-ctx.Done():
			return setupError{"enable", ctx.Err()}
		case <-ticker.C:
		}
	}
}
func (m *IME) Remove(ctx context.Context) error {
	m.maintenance.Lock()
	defer m.maintenance.Unlock()
	helper, _, err := m.paths()
	if err != nil {
		return err
	}
	// User configuration must be removed as the desktop user, before the
	// separate administrator transaction removes machine registration.
	if _, err = m.command(ctx, "disable"); err != nil {
		return err
	}
	if err = elevated(ctx, helper, []string{"remove"}); err != nil {
		return err
	}
	state, err := m.Query(ctx)
	if err != nil {
		return err
	}
	if state.Registered {
		return errors.New("Windows TSF removal was not observed")
	}
	return nil
}
func (m *IME) OpenSettings() error {
	command := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "ms-settings:typing")
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command.Run()
}
func (m *IME) ApplyKeyboard(settings application.KeyboardPreferences, key application.HoldKey) error {
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	return imebridge.WriteKeyboardSettings(filepath.Join(base, "Arion", "keyboard.json"), settings, key)
}
func (m *IME) UpdateAvailable() bool {
	build, err := m.validate()
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	state, err := m.Query(ctx)
	return err == nil && state.Registered && !state.current(build)
}

type shellExecuteInfo struct {
	Size       uint32
	Mask       uint32
	Window     uintptr
	Verb       *uint16
	File       *uint16
	Parameters *uint16
	Directory  *uint16
	Show       int32
	Instance   uintptr
	IDList     uintptr
	Class      *uint16
	ClassKey   uintptr
	HotKey     uint32
	Icon       uintptr
	Process    syscall.Handle
}

func elevated(ctx context.Context, helper string, args []string) error {
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, err := syscall.UTF16PtrFromString(helper)
	if err != nil {
		return err
	}
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = syscall.EscapeArg(arg)
	}
	parameters, err := syscall.UTF16PtrFromString(strings.Join(quoted, " "))
	if err != nil {
		return err
	}
	info := shellExecuteInfo{Mask: 0x40 | 0x100 | 0x400, Verb: verb, File: file, Parameters: parameters, Show: 0}
	info.Size = uint32(unsafe.Sizeof(info))
	ok, _, failure := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return fmt.Errorf("Windows TSF administrator approval: %w", failure)
	}
	if info.Process == 0 {
		return errors.New("Windows TSF installer returned no process")
	}
	defer syscall.CloseHandle(info.Process)
	for {
		result, err := syscall.WaitForSingleObject(info.Process, 100)
		if err != nil {
			return err
		}
		if result == syscall.WAIT_OBJECT_0 {
			var code uint32
			if err := syscall.GetExitCodeProcess(info.Process, &code); err != nil {
				return err
			}
			if code != 0 {
				if code == 75 {
					return updateDeferred{}
				}
				return fmt.Errorf("Windows TSF elevated helper failed (%d)", code)
			}
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
}
