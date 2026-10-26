package cloudpolish

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/internal/account"
	"github.com/AstraFoundry/republic"
)

// Credential is the one connected provider, kept as JSON in a single OS
// credential item. A ChatGPT sign-in keeps only its refresh token: access
// tokens are fetched when first needed and live in memory.
type Credential struct {
	Provider     string `json:"provider"`
	APIKey       string `json:"apiKey,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	AccountID    string `json:"accountId,omitempty"`
	Email        string `json:"email,omitempty"`
}

func (c Credential) valid() bool {
	if c.Provider == ChatGPT {
		return c.RefreshToken != "" && c.AccountID != ""
	}
	return Known(c.Provider) && c.APIKey != ""
}

// Secrets is the OS credential item; Get returns account.ErrNotFound when
// nothing is stored.
type Secrets interface {
	Put(string) error
	Get() (string, error)
	Delete() error
}

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
	// Failure is the last failure; a new attempt clears it.
	Failure Code
}

// Service is safe for concurrent use. It satisfies ports.Rewriter.
type Service struct {
	secrets  Secrets
	browser  func(string) error
	page     func() []byte
	client   *http.Client
	template prompt.Template
	// issuer and base replace the production endpoints in tests.
	issuer string
	base   string

	opening          sync.Mutex
	mu               sync.Mutex
	status           Status
	credential       Credential
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
func New(secrets Secrets, browser func(string) error, page func() []byte) *Service {
	return &Service{secrets: secrets, browser: browser, page: page, client: &http.Client{Transport: http.DefaultTransport},
		template: prompt.Live(), issuer: chatgptIssuer, status: Status{Phase: Disconnected}}
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

// Load reads the stored credential without touching the network.
func (s *Service) Load() {
	raw, err := s.secrets.Get()
	if errors.Is(err, account.ErrNotFound) {
		return
	}
	var c Credential
	if err == nil && (json.Unmarshal([]byte(raw), &c) != nil || !c.valid()) {
		err = errors.New("stored credential unreadable")
	}
	if err != nil {
		s.set(func(st *Status) { st.Failure = CodeStore })
		return
	}
	s.mu.Lock()
	s.credential = c
	s.mu.Unlock()
	s.set(func(st *Status) { *st = Status{Phase: Connected, Provider: c.Provider, Email: c.Email} })
}

// SetModel selects the model rewrites use.
func (s *Service) SetModel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = id
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

// connect stores c and makes it the connection; with conn nil the
// provider opens on first use.
func (s *Service) connect(ctx context.Context, generation uint64, c Credential, conn *connection) error {
	raw, err := json.Marshal(c)
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
	if err == nil {
		err = s.secrets.Put(string(raw))
	}
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
	var models []string
	if conn != nil {
		s.provider, models = conn.provider, conn.models
	}
	s.mu.Unlock()
	s.set(func(st *Status) { *st = Status{Phase: Connected, Provider: c.Provider, Email: c.Email, Models: models} })
	if conn == nil {
		s.ListModels()
	}
	return nil
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

// Disconnect forgets the connection and deletes its credential item.
func (s *Service) Disconnect() error {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.generation++
	s.closeLocked()
	s.credential = Credential{}
	err := s.secrets.Delete()
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
	if raw, err := json.Marshal(c); err == nil && s.secrets.Put(string(raw)) == nil {
		s.credential = c
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

// Rewrite satisfies ports.Rewriter. Only the text, the scene and the
// recalled terms are sent; context in req is never read.
func (s *Service) Rewrite(ctx context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
	s.mu.Lock()
	model := s.model
	s.mu.Unlock()
	if model == "" {
		return ports.RewriteResult{}, domain.Fail(domain.CodeModelMissing, "no cloud model chosen")
	}
	p, err := s.ready(ctx)
	if err != nil {
		return ports.RewriteResult{}, s.settle(err, CodeNetwork)
	}
	chat, err := p.GetModel(model)
	if err != nil {
		return ports.RewriteResult{}, fail(CodeModels, err)
	}
	var messages []republic.Message
	for _, t := range s.template.Turns(prompt.Input{Scene: req.Scene, Terms: req.Terms, Text: req.Text}) {
		switch t.Role {
		case "system":
			messages = append(messages, republic.System(t.Content))
		case "assistant":
			messages = append(messages, republic.Assistant(republic.Text(t.Content)))
		default:
			messages = append(messages, republic.User(republic.Text(t.Content)))
		}
	}
	response, err := chat.Chat(ctx, messages, nil)
	if err = s.settle(err, CodeNetwork); err != nil {
		return ports.RewriteResult{}, err
	}
	return ports.RewriteResult{Text: strings.TrimSpace(response.Text()), Template: "cloud+" + s.template.Version}, nil
}

// End satisfies ports.Rewriter; requests keep no session state.
func (s *Service) End(contracts.Identity) {}

var _ ports.Rewriter = (*Service)(nil)
