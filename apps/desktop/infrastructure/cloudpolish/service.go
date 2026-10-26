package cloudpolish

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/keystore"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/rewrite"
	"arion.local/desktop/internal/account"
	"github.com/AstraFoundry/republic"
)

// Credential is one provider's cloud polishing secret, kept as JSON in its
// own OS credential item (account cloud-polish/<provider>). A ChatGPT
// sign-in keeps only its refresh token: access tokens are fetched when
// first needed and live in memory. Model is the model last chosen for this
// provider; it is not a secret.
type Credential struct {
	Provider     string `json:"provider"`
	APIKey       string `json:"apiKey,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	AccountID    string `json:"accountId,omitempty"`
	Email        string `json:"email,omitempty"`
	Model        string `json:"model,omitempty"`
}

func (c Credential) valid() bool {
	if c.Provider == ChatGPT {
		return c.RefreshToken != "" && c.AccountID != ""
	}
	return Known(c.Provider) && c.APIKey != ""
}

// SecretStore is the OS credential catalog, one item per account name.
// Get returns account.ErrNotFound when that account is empty.
// credentialAccount holds only the active provider id. Each provider's
// secret lives at credentialAccountFor(provider). A build that stored the
// whole credential in credentialAccount is copied across on Load.
type SecretStore interface {
	Put(account, secret string) error
	Get(account string) (string, error)
	Delete(account string) error
}

func credentialAccountFor(provider string) string {
	return keystore.CloudPolishProviderAccount(provider)
}

func activeAccount() string { return keystore.CloudPolishAccount }

// Phase is the connection's lifecycle.
type Phase string

const (
	Disconnected Phase = "disconnected"
	// SigningIn waits for the ChatGPT sign-in in the browser.
	SigningIn Phase = "signing_in"
	// Verifying checks an API key by listing the provider's models.
	Verifying Phase = "verifying"
	Connected Phase = "connected"
)

// Status is a secret-free snapshot for the UI.
type Status struct {
	Phase Phase
	// Provider is the connected provider, or the one being connected.
	Provider string
	// Email is the ChatGPT account's.
	Email string
	// Models are the connected provider's model IDs once listed.
	Models  []string
	Listing bool
	// Model is the model last chosen for the active provider.
	Model string
	// Saved lists providers that still have a credential, including the
	// active one. Switching the active provider does not remove these.
	Saved []string
	// Failure is the last failure; a new attempt clears it.
	Failure Code
}

// Service is safe for concurrent use. It satisfies ports.Rewriter.
type Service struct {
	secrets SecretStore
	browser func(string) error
	page    func() []byte
	client  *http.Client
	// issuer and base replace the production endpoints in tests.
	issuer string
	base   string

	opening          sync.Mutex
	mu               sync.Mutex
	status           Status
	credential       Credential
	saved            map[string]Credential
	provider         *republic.Provider
	auth             *republic.TokenAuth
	model            string
	completionFormat string
	cancel           context.CancelFunc
	// generation invalidates work started before a connect or disconnect.
	generation uint64
	listeners  []func(Status)
}

// New prepares the service; it reads and sends nothing.
func New(secrets SecretStore, browser func(string) error, page func() []byte) *Service {
	return &Service{secrets: secrets, browser: browser, page: page, client: &http.Client{Transport: http.DefaultTransport},
		issuer: chatgptIssuer, status: Status{Phase: Disconnected}}
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
	s.status.Saved = s.savedIDs()
	status := s.status
	listeners := append([]func(Status){}, s.listeners...)
	s.mu.Unlock()
	for _, l := range listeners {
		l(status)
	}
}

func (s *Service) savedIDs() []string {
	if len(s.saved) == 0 {
		return nil
	}
	ids := make([]string, 0, len(s.saved))
	for id := range s.saved {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (s *Service) remember(c Credential) {
	if s.saved == nil {
		s.saved = map[string]Credential{}
	}
	s.saved[c.Provider] = c
}

// Load reads stored credentials without touching the network. A legacy
// single-item credential is copied to that provider's account, and the old
// account becomes a pointer at the active provider.
func (s *Service) Load() {
	saved, active, err := s.readCatalog()
	s.mu.Lock()
	s.saved = saved
	if active.valid() {
		// A model already chosen for this process (the preference applied
		// before Load) wins. Otherwise restore the model stored with the key.
		if s.model != "" && active.Model != s.model {
			active.Model = s.model
			s.remember(active)
			if raw, mErr := json.Marshal(active); mErr != nil {
				err = mErr
			} else if mErr = s.secrets.Put(credentialAccountFor(active.Provider), string(raw)); mErr != nil {
				err = mErr
			}
		}
		s.credential = active
		if s.model == "" {
			s.model = active.Model
		}
	} else {
		s.credential = Credential{}
	}
	model := s.model
	s.mu.Unlock()
	if !active.valid() {
		failure := Code("")
		if err != nil {
			failure = CodeStore
		}
		s.set(func(st *Status) { *st = Status{Phase: Disconnected, Failure: failure} })
		return
	}
	failure := Code("")
	if err != nil {
		failure = CodeStore
	}
	s.set(func(st *Status) {
		*st = Status{Phase: Connected, Provider: active.Provider, Email: active.Email, Model: model, Failure: failure}
	})
}

// SetModel selects the model rewrites use and remembers it on the active
// provider so a later switch can restore it.
func (s *Service) SetModel(id string) {
	s.mu.Lock()
	s.model = id
	var storeErr error
	if s.secrets != nil && s.credential.valid() && s.credential.Model != id {
		c := s.credential
		c.Model = id
		raw, err := json.Marshal(c)
		if err == nil {
			err = s.secrets.Put(credentialAccountFor(c.Provider), string(raw))
		}
		if err != nil {
			storeErr = err
		} else {
			s.credential = c
			s.remember(c)
		}
	}
	connected := s.credential.valid()
	s.mu.Unlock()
	if storeErr != nil {
		s.set(func(st *Status) { st.Failure = CodeStore })
		return
	}
	if connected {
		s.set(func(st *Status) { st.Model = id })
	}
}

// begin starts a connection attempt, cancelling any other, and returns its
// context and generation.
func (s *Service) begin(phase Phase, provider string) (context.Context, uint64) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.cancel = cancel
	s.generation++
	generation := s.generation
	s.mu.Unlock()
	s.set(func(st *Status) { st.Phase, st.Provider, st.Failure = phase, provider, "" })
	return ctx, generation
}

// end restores the previous connection after a failed attempt.
func (s *Service) end(generation uint64, err error) error {
	s.mu.Lock()
	if s.generation != generation {
		s.mu.Unlock()
		return err
	}
	s.cancel = nil
	c := s.credential
	s.mu.Unlock()
	s.set(func(st *Status) {
		*st = Status{Phase: Disconnected, Models: st.Models}
		if c.valid() {
			st.Phase, st.Provider, st.Email = Connected, c.Provider, c.Email
		} else {
			st.Models = nil
		}
		if code := CodeOf(err); code != CodeCancelled {
			st.Failure = code
		}
	})
	return err
}

// SignInChatGPT runs the browser sign-in and replaces the connection with
// it. It blocks until the callback, a cancel or the timeout.
func (s *Service) SignInChatGPT() error {
	ctx, generation := s.begin(SigningIn, ChatGPT)
	c, err := signInChatGPT(ctx, s.issuer, s.client, s.browser, s.page)
	if err != nil {
		return s.end(generation, err)
	}
	return s.connect(ctx, generation, c, nil)
}

// ConnectKey checks key by listing provider's models and, once it works,
// replaces the connection with it.
func (s *Service) ConnectKey(provider, key string) error {
	ctx, generation := s.begin(Verifying, provider)
	key = strings.TrimSpace(key)
	c := Credential{Provider: provider, APIKey: key}
	if provider == ChatGPT || !c.valid() {
		return s.end(generation, fail(CodeRejected, errors.New("unknown provider or empty key")))
	}
	p, err := open(c, nil, s.client, s.base)
	if err != nil {
		return s.end(generation, fail(CodeRejected, err))
	}
	models, err := listModels(ctx, p)
	if err != nil {
		_ = p.Close()
		return s.end(generation, classify(err, CodeRejected))
	}
	return s.connect(ctx, generation, c, &connection{provider: p, models: models})
}

type connection struct {
	provider *republic.Provider
	models   []string
}

// connect stores c and makes it the active provider. Other providers' items
// stay. With conn nil the provider opens on first use.
func (s *Service) connect(ctx context.Context, generation uint64, c Credential, conn *connection) error {
	// The item is written under mu so a newer attempt or a disconnect can
	// never be overwritten by this one.
	s.mu.Lock()
	if s.generation != generation {
		s.mu.Unlock()
		if conn != nil {
			_ = conn.provider.Close()
		}
		return fail(CodeCancelled, ctx.Err())
	}
	if c.Model == "" {
		if s.credential.Provider == "" || s.credential.Provider == c.Provider {
			c.Model = s.model
		} else if prev, ok := s.saved[c.Provider]; ok {
			c.Model = prev.Model
		}
	}
	err := s.writeActive(c)
	if err != nil {
		s.mu.Unlock()
		if conn != nil {
			_ = conn.provider.Close()
		}
		return s.end(generation, fail(CodeStore, err))
	}
	s.closeLocked()
	s.cancel = nil
	s.credential = c
	s.model = c.Model
	s.remember(c)
	var models []string
	if conn != nil {
		s.provider, models = conn.provider, conn.models
	}
	s.mu.Unlock()
	s.set(func(st *Status) {
		*st = Status{Phase: Connected, Provider: c.Provider, Email: c.Email, Models: models, Model: c.Model}
	})
	if conn == nil {
		s.ListModels()
	} else {
		s.adoptDefault(c.Provider, models)
	}
	return nil
}

// adoptDefault starts a provider with no chosen model on its default tier
// (DefaultModel), so a new connection polishes without another step.
func (s *Service) adoptDefault(provider string, models []string) {
	s.mu.Lock()
	chosen := s.model != "" || s.credential.Provider != provider
	s.mu.Unlock()
	if id := DefaultModel(provider, models); !chosen && id != "" {
		s.SetModel(id)
	}
}

// Activate makes a saved provider the active connection. It does not ask
// for the key again and does not delete any other provider's credential.
func (s *Service) Activate(provider string) error {
	s.mu.Lock()
	c, ok := s.saved[provider]
	if !ok || !c.valid() {
		s.mu.Unlock()
		return fail(CodeRejected, errors.New("no saved cloud credential"))
	}
	if s.credential.Provider == provider && s.credential.valid() && s.status.Phase == Connected && s.cancel == nil {
		s.mu.Unlock()
		return nil
	}
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	if err := s.writeActive(c); err != nil {
		s.mu.Unlock()
		s.set(func(st *Status) { st.Failure = CodeStore })
		return fail(CodeStore, err)
	}
	s.closeLocked()
	s.credential = c
	s.model = c.Model
	s.mu.Unlock()
	s.set(func(st *Status) {
		*st = Status{Phase: Connected, Provider: c.Provider, Email: c.Email, Model: c.Model}
	})
	s.ListModels()
	return nil
}

// writeActive stores c under its provider account and points the legacy
// account at that provider. Other saved credentials are written first so
// replacing the legacy item cannot drop a key that still lives only there.
// Callers hold mu.
func (s *Service) writeActive(c Credential) error {
	for id, item := range s.saved {
		if id == c.Provider {
			continue
		}
		raw, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if err := s.secrets.Put(credentialAccountFor(id), string(raw)); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := s.secrets.Put(credentialAccountFor(c.Provider), string(raw)); err != nil {
		return err
	}
	pointer, err := json.Marshal(Credential{Provider: c.Provider})
	if err != nil {
		return err
	}
	return s.secrets.Put(activeAccount(), string(pointer))
}

// readCatalog loads every provider item and the legacy account. A valid
// credential still stored in the legacy account is copied to its provider
// item, then the legacy account is rewritten as a pointer.
func (s *Service) readCatalog() (map[string]Credential, Credential, error) {
	saved := map[string]Credential{}
	var readErr error
	for _, id := range knownProviders() {
		c, err := s.readAccount(credentialAccountFor(id))
		if errors.Is(err, account.ErrNotFound) {
			continue
		}
		if err != nil {
			readErr = err
			continue
		}
		if c.Provider != id {
			readErr = errors.New("stored credential unreadable")
			continue
		}
		saved[id] = c
	}
	raw, err := s.secrets.Get(activeAccount())
	if errors.Is(err, account.ErrNotFound) {
		return saved, Credential{}, readErr
	}
	if err != nil {
		return saved, Credential{}, err
	}
	var c Credential
	if json.Unmarshal([]byte(raw), &c) != nil || !Known(c.Provider) {
		return saved, Credential{}, errors.New("stored credential unreadable")
	}
	if !c.valid() {
		if active, ok := saved[c.Provider]; ok {
			return saved, active, readErr
		}
		return saved, Credential{}, readErr
	}
	if _, ok := saved[c.Provider]; !ok {
		saved[c.Provider] = c
		item, mErr := json.Marshal(c)
		if mErr != nil {
			return saved, c, mErr
		}
		if mErr = s.secrets.Put(credentialAccountFor(c.Provider), string(item)); mErr != nil {
			return saved, c, mErr
		}
	}
	pointer, mErr := json.Marshal(Credential{Provider: c.Provider})
	if mErr != nil {
		return saved, saved[c.Provider], mErr
	}
	if mErr = s.secrets.Put(activeAccount(), string(pointer)); mErr != nil {
		return saved, saved[c.Provider], mErr
	}
	return saved, saved[c.Provider], readErr
}

func (s *Service) readAccount(accountName string) (Credential, error) {
	raw, err := s.secrets.Get(accountName)
	if err != nil {
		return Credential{}, err
	}
	var c Credential
	if json.Unmarshal([]byte(raw), &c) != nil || !c.valid() {
		return Credential{}, errors.New("stored credential unreadable")
	}
	return c, nil
}

func (s *Service) closeLocked() {
	if s.provider != nil {
		_ = s.provider.Close()
	}
	s.provider, s.auth = nil, nil
}

// Cancel stops a sign-in or key check in flight.
func (s *Service) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// Disconnect forgets the active provider and deletes only its credential.
// Every other provider's key stays stored.
func (s *Service) Disconnect() error {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	s.closeLocked()
	provider := s.credential.Provider
	s.credential = Credential{}
	s.model = ""
	var err error
	if provider != "" {
		if delErr := s.secrets.Delete(credentialAccountFor(provider)); delErr != nil {
			err = delErr
		} else {
			delete(s.saved, provider)
		}
	}
	if delErr := s.secrets.Delete(activeAccount()); delErr != nil && err == nil {
		err = delErr
	}
	s.mu.Unlock()
	s.set(func(st *Status) {
		*st = Status{Phase: Disconnected}
		if err != nil {
			st.Failure = CodeStore
		}
	})
	return err
}

// ListModels refreshes the connected provider's model list.
func (s *Service) ListModels() {
	s.mu.Lock()
	generation := s.generation
	connected := s.credential.valid()
	s.mu.Unlock()
	if !connected {
		return
	}
	s.set(func(st *Status) { st.Listing, st.Failure = true, "" })
	p, err := s.ready(context.Background())
	var models []string
	if err == nil {
		models, err = listModels(context.Background(), p)
	}
	err = s.settle(err, CodeModels)
	s.mu.Lock()
	current := s.generation == generation
	s.mu.Unlock()
	if !current {
		return
	}
	s.set(func(st *Status) {
		st.Listing = false
		if err != nil {
			st.Failure = CodeOf(err)
			return
		}
		st.Models = models
	})
	if err == nil {
		s.mu.Lock()
		provider := s.credential.Provider
		s.mu.Unlock()
		s.adoptDefault(provider, models)
	}
}

// ready opens the provider on first use. A ChatGPT connection trades its
// refresh token for an access token here. Opening is serialized apart from
// mu, so status reads never wait on the network.
func (s *Service) ready(ctx context.Context) (*republic.Provider, error) {
	s.opening.Lock()
	defer s.opening.Unlock()
	s.mu.Lock()
	p, c, generation := s.provider, s.credential, s.generation
	s.mu.Unlock()
	if p != nil {
		return p, nil
	}
	if !c.valid() {
		return nil, domain.Fail(domain.CodeModelMissing, "no cloud provider connected")
	}
	var auth *republic.TokenAuth
	if c.Provider == ChatGPT {
		tok, err := refreshChatGPT(ctx, s.issuer, s.client, c.RefreshToken)
		if err != nil {
			return nil, err
		}
		auth, err = republic.NewCodexAuth(map[string]any{"access_token": tok.AccessToken, "refresh_token": tok.RefreshToken, "id_token": tok.IDToken}, c.AccountID)
		if err != nil {
			return nil, fail(CodeRejected, err)
		}
		auth.HTTPClient = s.client
	}
	p, err := open(c, auth, s.client, s.base)
	if err != nil {
		return nil, fail(CodeRejected, err)
	}
	s.mu.Lock()
	if s.generation != generation {
		s.mu.Unlock()
		_ = p.Close()
		return nil, fail(CodeCancelled, errors.New("connection replaced"))
	}
	s.provider, s.auth = p, auth
	s.mu.Unlock()
	s.persist()
	return p, nil
}

// persist stores a rotated ChatGPT refresh token. It writes under mu, so a
// disconnect cannot be undone by a request that was in flight.
func (s *Service) persist() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.auth == nil {
		return
	}
	refresh, _ := s.auth.Token()["refresh_token"].(string)
	if refresh == "" || refresh == s.credential.RefreshToken {
		return
	}
	c := s.credential
	c.RefreshToken = refresh
	if raw, err := json.Marshal(c); err == nil && s.secrets.Put(credentialAccountFor(c.Provider), string(raw)) == nil {
		s.credential = c
		s.remember(c)
	}
}

// settle records a rotated token after a request and classifies its error;
// a refused credential is shown so the user can reconnect.
func (s *Service) settle(err error, otherwise Code) error {
	s.persist()
	if err == nil || domain.CodeOf(err) != "" {
		return err
	}
	err = classify(err, otherwise)
	if CodeOf(err) == CodeRejected {
		s.set(func(st *Status) { st.Failure = CodeRejected })
	}
	return err
}

// classify maps a Republic failure to a Code.
func classify(err error, otherwise Code) error {
	if CodeOf(err) != "" {
		return err
	}
	var r *republic.Error
	if errors.As(err, &r) {
		switch {
		case r.Kind == republic.ErrAuthentication || r.StatusCode == http.StatusUnauthorized || r.StatusCode == http.StatusForbidden:
			return fail(CodeRejected, err)
		case r.Kind == republic.ErrConnection || r.Kind == republic.ErrTimeout:
			return fail(CodeNetwork, err)
		}
	}
	return fail(otherwise, err)
}

// Ready reports a connection and a chosen model.
func (s *Service) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.credential.valid() && s.model != ""
}

// Rewrite uses the same staged pipeline as the local worker. The coordinator
// supplies only the context sources enabled by the user's preferences.
func (s *Service) Rewrite(ctx context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
	if req.Prefill {
		return ports.RewriteResult{}, errors.New("cloud prefill is unsupported")
	}
	chat, err := s.completionModel(ctx)
	if err != nil {
		return ports.RewriteResult{}, err
	}
	pipeline, err := rewrite.Load()
	if err != nil {
		return ports.RewriteResult{}, err
	}
	result, err := pipeline.Run(ctx, engine.Request{Text: req.Text, Scene: req.Scene, Punctuation: domain.Punctuation(req.Punctuation), TargetLanguage: domain.TranslationLanguage(req.TargetLanguage), Terms: req.Terms,
		Context: req.Context, PreviousText: req.PreviousText, Live: req.Live},
		func(ctx context.Context, request prompt.ChatRequest, _ bool) (engine.Result, error) {
			return s.completeChat(ctx, chat, request)
		})
	out := ports.RewriteResult{Text: result.Text, Cleanup: result.Cleanup, Template: "cloud+" + result.Template,
		Truncated: result.Truncated, InputTokens: result.InputTokens, OutputTokens: result.OutputTokens,
		ReusedTokens: result.ReusedTokens, PrefillMs: result.PrefillMs, DecodeMs: result.DecodeMs}
	if result.Cancelled {
		return out, domain.Fail(domain.CodeCancelled, "rewrite cancelled")
	}
	return out, err
}

// End satisfies ports.Rewriter; requests keep no session state.
func (s *Service) End(contracts.Identity) {}

var _ ports.Rewriter = (*Service)(nil)
