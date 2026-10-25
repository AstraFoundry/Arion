// Package rewriting owns the optional local rewrite capability: the
// bundled LLM worker and runtime, provisioning of the catalog model and the
// worker's lifecycle. It satisfies the core Rewriter port and reports
// MODEL_MISSING whenever no verified model is running.
package rewriting

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/models"
)

// DefaultModel is the catalog rewriter the live pipeline uses; the
// benchmark report records why it was chosen.
const DefaultModel = "qwen3.5-2b-q4-k-m-metal"

// Timeout gives the Windows CPU pipeline a bounded final-request budget. Zero
// preserves the coordinator's default on other platforms; cloud owns its limit.
func Timeout(platform string) time.Duration {
	if platform == "windows-amd64" {
		return 30 * time.Second
	}
	return 0
}

// State is the capability's user-visible readiness.
type State string

const (
	// Unsupported: this build ships no runtime for the platform.
	Unsupported  State = "unsupported"
	Off          State = "off"
	Provisioning State = "provisioning"
	Loading      State = "loading"
	Ready        State = "ready"
	Failed       State = "failed"
)

// Status is a content-free snapshot for the UI.
type Status struct {
	State     State
	Phase     provision.Phase
	Done      int64
	Total     int64
	Failure   domain.ErrorCode
	Installed bool
	SizeBytes int64
	Name      string
}

// Locate returns the bundled LLM directory holding the worker and the
// GGUF runtime, or "" when this build ships none. Packaged apps flatten
// platform resources; development runs keep them per platform.
func Locate(resources string) string {
	for _, dir := range []string{filepath.Join(resources, "llm"), filepath.Join(resources, runtime.GOOS+"-"+runtime.GOARCH, "llm")} {
		if fileExists(filepath.Join(dir, workerName())) && fileExists(filepath.Join(dir, runtimeName())) {
			return dir
		}
	}
	return ""
}

func workerName() string {
	if runtime.GOOS == "windows" {
		return "arion-llm.exe"
	}
	return "arion-llm"
}

func runtimeName() string {
	if runtime.GOOS == "windows" {
		return "llama-server.exe"
	}
	return "llama-server"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Service is safe for concurrent use.
type Service struct {
	store   *provision.Store
	model   models.Rewriter
	bundled string
	threads int
	diag    io.Writer

	mu        sync.Mutex
	status    Status
	client    *client.Client
	cancel    context.CancelFunc
	listeners []func(Status)
}

// New prepares the service; it starts nothing.
func New(catalog models.Catalog, store *provision.Store, bundled string, threads int, platform string) *Service {
	s := &Service{store: store, bundled: bundled, threads: threads, diag: io.Discard}
	model, ok := catalog.Rewriter(DefaultModel)
	supported := ok && bundled != ""
	if ok {
		supported = supported && contains(model.Platforms, platform)
		s.model = model
	}
	s.status = Status{State: Off, SizeBytes: model.SizeBytes(), Name: model.Name}
	if !supported {
		s.status.State = Unsupported
	}
	return s
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// OnStatus registers a listener called on every status change.
func (s *Service) OnStatus(listener func(Status)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Service) set(update func(*Status)) {
	s.mu.Lock()
	update(&s.status)
	status := s.status
	listeners := append([]func(Status){}, s.listeners...)
	s.mu.Unlock()
	for _, l := range listeners {
		l(status)
	}
}

// Refresh records whether the model is installed, without starting it.
func (s *Service) Refresh() {
	if s.Status().State == Unsupported {
		return
	}
	_, err := s.store.InstalledRewriter(s.model)
	s.set(func(st *Status) { st.Installed = err == nil })
}

// Enable starts an installed model without downloading. A missing model stays
// off until the user explicitly requests Download.
func (s *Service) Enable(ctx context.Context) error {
	return s.enable(ctx, false)
}

// Download provisions the model with the user's explicit consent, then starts
// the worker. It blocks until ready, cancelled or failed.
func (s *Service) Download(ctx context.Context) error {
	return s.enable(ctx, true)
}

func (s *Service) enable(ctx context.Context, download bool) error {
	s.mu.Lock()
	switch {
	case s.status.State == Unsupported:
		s.mu.Unlock()
		return domain.Fail(domain.CodeModelUnsupported, "rewrite runtime not bundled for this platform")
	case s.client != nil && s.client.Alive():
		s.mu.Unlock()
		return nil
	case s.cancel != nil:
		s.mu.Unlock()
		return nil // already enabling
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.cancel = nil
		s.mu.Unlock()
		cancel()
	}()
	dir, err := s.store.InstalledRewriter(s.model)
	if download {
		s.set(func(st *Status) { st.State, st.Failure = Provisioning, "" })
		dir, err = s.store.EnsureRewriter(ctx, s.model, func(_ string, phase provision.Phase, done, total int64) {
			s.set(func(st *Status) { st.Phase, st.Done, st.Total = phase, done, total })
		})
	} else if domain.CodeOf(err) == domain.CodeModelMissing {
		s.set(func(st *Status) { st.State, st.Failure, st.Installed = Off, "", false })
		return nil
	}
	if err != nil {
		return s.fail(err)
	}
	s.set(func(st *Status) { st.State, st.Installed = Loading, true })
	c, _, err := client.Start(ctx, client.Config{Worker: filepath.Join(s.bundled, workerName()), Runtime: s.bundled, Model: dir,
		Threads: s.threads, MaxLength: 4096, MaxNewTokens: 256}, s.diag)
	if err != nil {
		return s.fail(err)
	}
	s.mu.Lock()
	if ctx.Err() != nil {
		s.mu.Unlock()
		c.Close()
		s.set(func(st *Status) { st.State = Off })
		return domain.Fail(domain.CodeCancelled, "rewrite disabled")
	}
	s.client = c
	s.mu.Unlock()
	s.set(func(st *Status) { st.State = Ready })
	return nil
}

func (s *Service) fail(err error) error {
	code := domain.CodeOf(err)
	if code == domain.CodeCancelled {
		s.set(func(st *Status) { st.State = Off })
		return err
	}
	if code == "" {
		code = domain.CodeModelUnsupported
	}
	s.set(func(st *Status) { st.State, st.Failure = Failed, code })
	return err
}

// Disable stops provisioning or the worker; the model stays installed.
func (s *Service) Disable() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	c := s.client
	s.client = nil
	unsupported := s.status.State == Unsupported
	s.mu.Unlock()
	if c != nil {
		c.Close()
	}
	if !unsupported {
		s.set(func(st *Status) { st.State = Off })
	}
}

// Ready reports a running worker; the coordinator only rewrites when true.
func (s *Service) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client != nil && s.client.Alive()
}

// Rewrite satisfies ports.Rewriter.
func (s *Service) Rewrite(ctx context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
	s.mu.Lock()
	c := s.client
	s.mu.Unlock()
	if c == nil {
		return ports.RewriteResult{}, domain.Fail(domain.CodeModelMissing, "rewrite model not running")
	}
	result, err := c.Rewrite(ctx, req)
	if err != nil && !c.Alive() {
		// The worker died: report it once instead of failing every session.
		s.mu.Lock()
		if s.client == c {
			s.client = nil
		}
		s.mu.Unlock()
		s.set(func(st *Status) { st.State, st.Failure = Failed, domain.CodeModelUnsupported })
	}
	return result, err
}

// End satisfies ports.Rewriter.
func (s *Service) End(id contracts.Identity) {
	s.mu.Lock()
	c := s.client
	s.mu.Unlock()
	if c != nil {
		c.End(id)
	}
}

var _ ports.Rewriter = (*Service)(nil)
