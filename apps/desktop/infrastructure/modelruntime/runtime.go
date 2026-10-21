// Package modelruntime selects, provisions and keeps the recognition bundle
// resident, and exposes it through the core recognition ports.
package modelruntime

import (
	"context"
	"runtime"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/asr"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/models"
)

// Platform is the catalog/evaluation hardware tag of this build.
func Platform() string { return runtime.GOOS + "-" + runtime.GOARCH }

// Options maps catalog bundles to the domain selection inputs.
func Options(catalog models.Catalog) []domain.ModelOption {
	var options []domain.ModelOption
	for _, bundle := range catalog.Bundles {
		streaming, ok := catalog.Artifact(bundle.Streaming)
		if !ok {
			continue
		}
		option := domain.ModelOption{ID: bundle.ID, Adopted: bundle.Adopted, Platforms: streaming.Platforms, Languages: streaming.Languages, SizeBytes: catalog.DownloadBytes(bundle)}
		if bundle.Finalizer != "" {
			finalizer, _ := catalog.Artifact(bundle.Finalizer)
			option.Platforms = intersect(option.Platforms, finalizer.Platforms)
			option.Languages = intersect(option.Languages, finalizer.Languages)
		}
		for _, e := range bundle.Evaluations {
			option.Evaluation = append(option.Evaluation, domain.ModelEvaluation{
				RunID: e.RunID, Hardware: e.Hardware, ErrorRates: e.ErrorRates, StopToReviewP95Ms: e.StopToReviewP95Ms,
				FirstPartialP95Ms: e.FirstPartialP95Ms, RealTimeFactor: e.RealTimeFactor, Observations: e.Observations,
			})
		}
		options = append(options, option)
	}
	return options
}

func intersect(a, b []string) []string {
	var out []string
	for _, x := range a {
		for _, y := range b {
			if x == y {
				out = append(out, x)
				break
			}
		}
	}
	return out
}

// SelectDefault applies the domain rule to the embedded catalog.
func SelectDefault(catalog models.Catalog) (models.Bundle, domain.Selection, error) {
	selection, err := domain.SelectDefaultModel(Options(catalog), domain.DefaultSelectionPolicy(Platform()))
	if err != nil {
		return models.Bundle{}, domain.Selection{}, err
	}
	bundle, _ := catalog.Bundle(selection.Option.ID)
	return bundle, selection, nil
}

// SpecFor builds the runtime file layout of a verified artifact directory.
func SpecFor(artifact models.Artifact, dir string, bundle models.Bundle, threads int) asr.Spec {
	files := map[string]string{}
	for role, file := range artifact.Files {
		files[role] = provision.FileName(file)
	}
	return asr.Spec{Kind: asr.Kind(artifact.Kind), Directory: dir, Files: files,
		SampleRate: artifact.Audio.SampleRate, Threads: threads, EndpointSilence: float32(bundle.EndpointMs) / 1000}
}

// State is the user-visible model readiness.
type State string

const (
	StateUnselected   State = "unselected"
	StateProvisioning State = "provisioning"
	StateLoading      State = "loading"
	StateReady        State = "ready"
	StateFailed       State = "failed"
)

// Status is a content-free snapshot for the UI.
type Status struct {
	State    State
	Bundle   string
	Artifact string
	Phase    provision.Phase
	Done     int64
	Total    int64
	Failure  domain.ErrorCode
}

// Manager owns the resident bundle. It implements ports.RecognizerFactory
// and ports.Finalizer; models are swapped only while no session uses them.
type Manager struct {
	catalog models.Catalog
	store   *provision.Store
	threads int

	mu        sync.Mutex
	idle      *sync.Cond
	active    int
	streaming *asr.Streaming
	offline   *asr.Offline
	bundle    models.Bundle
	status    Status
	listeners []func(Status)
}

func NewManager(catalog models.Catalog, store *provision.Store, threads int) *Manager {
	m := &Manager{catalog: catalog, store: store, threads: threads, status: Status{State: StateUnselected}}
	m.idle = sync.NewCond(&m.mu)
	return m
}

// OnStatus registers a status listener; it is called off the UI thread.
func (m *Manager) OnStatus(listener func(Status)) {
	m.mu.Lock()
	m.listeners = append(m.listeners, listener)
	status := m.status
	m.mu.Unlock()
	listener(status)
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *Manager) Bundle() models.Bundle {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bundle
}

func (m *Manager) setStatus(update func(*Status)) {
	m.mu.Lock()
	update(&m.status)
	status := m.status
	listeners := make([]func(Status), len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.Unlock()
	for _, listener := range listeners {
		listener(status)
	}
}

// Installed reports whether every artifact of a bundle verifies locally.
func (m *Manager) Installed(bundle models.Bundle) bool {
	for _, id := range m.artifactIDs(bundle) {
		artifact, _ := m.catalog.Artifact(id)
		if _, err := m.store.Installed(artifact); err != nil {
			return false
		}
	}
	return true
}

func (m *Manager) artifactIDs(bundle models.Bundle) []string {
	ids := []string{bundle.Streaming}
	if bundle.Finalizer != "" {
		ids = append(ids, bundle.Finalizer)
	}
	return ids
}

// Activate provisions (downloading only what is missing), verifies and
// loads a bundle, then swaps it in once no session holds the old one.
func (m *Manager) Activate(ctx context.Context, bundle models.Bundle) error {
	m.setStatus(func(s *Status) {
		*s = Status{State: StateProvisioning, Bundle: bundle.ID, Total: m.catalog.DownloadBytes(bundle)}
	})
	dirs := map[string]string{}
	var completed int64
	for _, id := range m.artifactIDs(bundle) {
		artifact, ok := m.catalog.Artifact(id)
		if !ok {
			return m.failed(bundle, domain.Fail(domain.CodeModelUnsupported, "unknown artifact"))
		}
		dir, err := m.store.Ensure(ctx, artifact, func(id string, phase provision.Phase, done, total int64) {
			m.setStatus(func(s *Status) { s.Artifact, s.Phase, s.Done = id, phase, completed+done })
		})
		if err != nil {
			return m.failed(bundle, err)
		}
		completed += artifact.Source.SizeBytes
		dirs[id] = dir
	}
	m.setStatus(func(s *Status) { s.State, s.Done = StateLoading, completed })
	streamingArtifact, _ := m.catalog.Artifact(bundle.Streaming)
	streaming, err := asr.LoadStreamingContext(ctx, SpecFor(streamingArtifact, dirs[bundle.Streaming], bundle, m.threads))
	if err != nil {
		return m.failed(bundle, err)
	}
	var offline *asr.Offline
	if bundle.Finalizer != "" {
		finalizerArtifact, _ := m.catalog.Artifact(bundle.Finalizer)
		offline, err = asr.LoadOffline(SpecFor(finalizerArtifact, dirs[bundle.Finalizer], bundle, m.threads))
		if err != nil {
			streaming.Close()
			return m.failed(bundle, err)
		}
	}
	m.mu.Lock()
	for m.active > 0 {
		m.idle.Wait()
	}
	oldStreaming, oldOffline := m.streaming, m.offline
	m.streaming, m.offline, m.bundle = streaming, offline, bundle
	m.mu.Unlock()
	if oldStreaming != nil {
		oldStreaming.Close()
	}
	if oldOffline != nil {
		oldOffline.Close()
	}
	m.setStatus(func(s *Status) { *s = Status{State: StateReady, Bundle: bundle.ID, Done: completed, Total: completed} })
	return nil
}

func (m *Manager) failed(bundle models.Bundle, err error) error {
	code := domain.CodeOf(err)
	if code == "" {
		code = domain.CodeProvisioningFailed
	}
	m.setStatus(func(s *Status) { s.State, s.Bundle, s.Failure = StateFailed, bundle.ID, code })
	return err
}

// NewRecognizer satisfies ports.RecognizerFactory.
func (m *Manager) NewRecognizer(ctx context.Context, id contracts.Identity) (ports.Recognizer, error) {
	m.mu.Lock()
	streaming := m.streaming
	if streaming == nil {
		m.mu.Unlock()
		return nil, domain.Fail(domain.CodeModelMissing, "no verified model loaded")
	}
	m.active++
	m.mu.Unlock()
	recognizer, err := streaming.NewRecognizer(ctx, id)
	if err != nil {
		m.release()
		return nil, err
	}
	return &tracked{Recognizer: recognizer, release: m.release}, nil
}

func (m *Manager) release() {
	m.mu.Lock()
	m.active--
	if m.active == 0 {
		m.idle.Broadcast()
	}
	m.mu.Unlock()
}

type tracked struct {
	ports.Recognizer
	once    sync.Once
	release func()
}

func (t *tracked) Close() error {
	err := t.Recognizer.Close()
	t.once.Do(t.release)
	return err
}

// Finalize satisfies ports.Finalizer. A bundle without a declared acoustic
// finalizer returns the streaming draft as its final hypothesis.
func (m *Manager) Finalize(ctx context.Context, request ports.FinalizeRequest) (string, error) {
	m.mu.Lock()
	offline := m.offline
	m.mu.Unlock()
	if offline == nil {
		return request.Draft, nil
	}
	return offline.Finalize(ctx, request)
}

// LatinCase reports how the active streaming artifact writes Latin text.
func (m *Manager) LatinCase() domain.LatinCase {
	m.mu.Lock()
	bundle := m.bundle
	m.mu.Unlock()
	artifactID := bundle.Streaming
	if bundle.Finalizer != "" {
		artifactID = bundle.Finalizer
	}
	if artifact, ok := m.catalog.Artifact(artifactID); ok && artifact.LatinCase == string(domain.LatinCaseUpper) {
		return domain.LatinCaseUpper
	}
	return domain.LatinCaseNatural
}

// Close releases resident native worker processes after sessions have stopped.
func (m *Manager) Close() {
	m.mu.Lock()
	streaming, offline := m.streaming, m.offline
	m.streaming, m.offline = nil, nil
	m.mu.Unlock()
	if streaming != nil {
		streaming.Close()
	}
	if offline != nil {
		offline.Close()
	}
}

// ComputeTime forwards infrastructure timing without widening the domain port.
func (t *tracked) ComputeTime() time.Duration {
	if recorder, ok := t.Recognizer.(interface{ ComputeTime() time.Duration }); ok {
		return recorder.ComputeTime()
	}
	return 0
}
