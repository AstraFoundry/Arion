package keyboardmodel

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/desktop/infrastructure/rewriting"
	"arion.local/models"
)

// Service owns a separately provisioned fast worker. Lifecycle calls are
// serialized by the bootstrap resource queue; inference may run concurrently.
type Service struct {
	mu         sync.Mutex
	worker     *Worker
	model      models.FileModel
	store      *provision.Store
	executable string
	status     rewriting.Status
	changed    func(rewriting.Status)
	suspended  bool
}

func Locate(resources string) string {
	name := "arion-keyboard-ranker"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, dir := range []string{resources, filepath.Join(resources, runtime.GOOS+"-"+runtime.GOARCH)} {
		file := filepath.Join(dir, "keyboard-ranker", name)
		if stat, err := os.Stat(file); err == nil && !stat.IsDir() {
			return file
		}
	}
	return ""
}
func New(catalog models.Catalog, store *provision.Store, executable string) *Service {
	model, ok := catalog.KeyboardRanker(ModelID)
	state := rewriting.Off
	if !ok || executable == "" {
		state = rewriting.Unsupported
	}
	return &Service{store: store, model: model, executable: executable, status: rewriting.Status{State: state, Name: model.Name, SizeBytes: model.SizeBytes()}}
}
func (s *Service) OnStatus(changed func(rewriting.Status)) {
	s.mu.Lock()
	s.changed = changed
	s.mu.Unlock()
}
func (s *Service) Status() rewriting.Status { s.mu.Lock(); defer s.mu.Unlock(); return s.status }
func (s *Service) set(update func(*rewriting.Status)) {
	s.mu.Lock()
	update(&s.status)
	status, changed := s.status, s.changed
	s.mu.Unlock()
	if changed != nil {
		changed(status)
	}
}
func (s *Service) Disable() {
	s.mu.Lock()
	worker := s.worker
	s.worker = nil
	s.mu.Unlock()
	if worker != nil {
		worker.Close()
	}
	s.set(func(st *rewriting.Status) {
		if st.State != rewriting.Unsupported {
			st.State = rewriting.Off
		}
	})
}
func (s *Service) Enable(ctx context.Context, download bool) error {
	if s.Status().State == rewriting.Unsupported {
		return domain.Fail(domain.CodeModelUnsupported, "keyboard runtime not bundled")
	}
	dir, err := s.store.InstalledRewriter(s.model)
	if download {
		s.set(func(st *rewriting.Status) { st.State, st.Failure = rewriting.Provisioning, "" })
		dir, err = s.store.EnsureRewriter(ctx, s.model, func(_ string, phase provision.Phase, done, total int64) {
			s.set(func(st *rewriting.Status) { st.Phase, st.Done, st.Total = phase, done, total })
		})
	} else if domain.CodeOf(err) == domain.CodeModelMissing {
		s.set(func(st *rewriting.Status) { st.State, st.Installed = rewriting.Off, false })
		return nil
	}
	if err == nil {
		s.set(func(st *rewriting.Status) { st.State, st.Installed = rewriting.Loading, true })
		load, cancel := context.WithTimeout(ctx, 5*time.Second)
		var worker *Worker
		worker, err = Open(load, s.executable, filepath.Join(dir, "sentence-model.safetensors"))
		cancel()
		if err == nil {
			s.mu.Lock()
			old := s.worker
			s.worker = worker
			s.mu.Unlock()
			if old != nil {
				old.Close()
			}
			s.set(func(st *rewriting.Status) { st.State, st.Failure = rewriting.Ready, "" })
			return nil
		}
	}
	s.set(func(st *rewriting.Status) { st.State, st.Failure = rewriting.Failed, domain.CodeModelUnsupported })
	return err
}
func (s *Service) SuspendKeyboard(suspended bool) {
	s.mu.Lock()
	s.suspended = suspended
	s.mu.Unlock()
}
func (s *Service) RankKeyboard(ctx context.Context, req contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	s.mu.Lock()
	worker, suspended := s.worker, s.suspended
	s.mu.Unlock()
	if worker == nil || suspended {
		return contracts.KeyboardRankResult{}, domain.Fail(domain.CodeModelMissing, "keyboard model unavailable")
	}
	out, err := worker.RankKeyboard(ctx, req)
	if err != nil && ctx.Err() == nil && !worker.Alive() {
		s.set(func(st *rewriting.Status) { st.State, st.Failure = rewriting.Failed, domain.CodeModelUnsupported })
	}
	return out, err
}
