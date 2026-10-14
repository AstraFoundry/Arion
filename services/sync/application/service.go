package application

import (
	"context"
	"errors"
	"time"

	"arion.local/sync/domain"
)

// LocalScope is the single partition used when AccountGate is noop.
const LocalScope = "local"

// Logger records content-free operational events.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Options wires a relay service. Nil collaborators are rejected.
type Options struct {
	Store     domain.Beginner
	Directory Directory
	Accounts  Accounts
	Gate      Gate
	Events    EventVerifier
	Limits    domain.Limits
	CursorTTL time.Duration
	Skew      time.Duration
	PageSize  int
	Now       func() time.Time
	Log       Logger
}

// Service orchestrates device authentication and revision policy. It asks the
// gate for a scope and does not inspect tokens itself.
type Service struct {
	store     domain.Beginner
	directory Directory
	accounts  Accounts
	gate      Gate
	events    EventVerifier
	limits    domain.Limits
	cursorTTL time.Duration
	skew      time.Duration
	pageSize  int
	now       func() time.Time
	log       Logger
}

// New constructs a service. Directory and Accounts may be nil only when the
// gate mode is noop and security events are not served.
func New(opt Options) (*Service, error) {
	if opt.Store == nil || opt.Gate == nil || opt.Now == nil || opt.Log == nil {
		return nil, errors.New("sync service is incomplete")
	}
	if opt.Limits.MaxBatch <= 0 || opt.CursorTTL <= 0 || opt.Skew <= 0 || opt.PageSize <= 0 {
		return nil, errors.New("sync service limits are invalid")
	}
	if opt.Gate.Mode() == "oidc" && (opt.Directory == nil || opt.Accounts == nil || opt.Events == nil) {
		return nil, errors.New("oidc gate requires a directory, account store, and event verifier")
	}
	return &Service{
		store:     opt.Store,
		directory: opt.Directory,
		accounts:  opt.Accounts,
		gate:      opt.Gate,
		events:    opt.Events,
		limits:    opt.Limits,
		cursorTTL: opt.CursorTTL,
		skew:      opt.Skew,
		pageSize:  opt.PageSize,
		now:       opt.Now,
		log:       opt.Log,
	}, nil
}

// Mode returns the configured account gate.
func (s *Service) Mode() string { return s.gate.Mode() }

func (s *Service) authorize(ctx context.Context, header string) (Scope, error) {
	return s.gate.Authorize(ctx, header)
}

// DeviceProof is the signed request envelope. Body is the raw bytes that were
// hashed into the signature.
type DeviceProof struct {
	DeviceID  string
	Timestamp time.Time
	Signature []byte
	Method    string
	Path      string
	Body      []byte
}

func (s *Service) withinSkew(ts time.Time) bool {
	now := s.now()
	return !ts.Before(now.Add(-s.skew)) && !ts.After(now.Add(s.skew))
}

func (s *Service) authenticate(ctx context.Context, tx domain.Tx, scope Scope, proof DeviceProof) (domain.Device, error) {
	if !domain.ValidID(proof.DeviceID) || !s.withinSkew(proof.Timestamp) {
		return domain.Device{}, domain.Fail(domain.CodeUnauthorized)
	}
	device, err := tx.Device(ctx, proof.DeviceID)
	if err != nil {
		if domain.IsCode(err, domain.CodeNotFound) {
			return domain.Device{}, domain.Fail(domain.CodeUnauthorized)
		}
		return domain.Device{}, err
	}
	if device.ScopeID != scope.ID {
		return domain.Device{}, domain.Fail(domain.CodeScopeMismatch)
	}
	if device.Status != domain.DeviceActive {
		return domain.Device{}, domain.Fail(domain.CodeDeviceRevoked)
	}
	message := domain.CanonicalRequest(proof.DeviceID, proof.Timestamp.UnixMilli(), proof.Method, proof.Path, proof.Body)
	if err := domain.VerifySignature(device.PublicKey, message, proof.Signature); err != nil {
		return domain.Device{}, err
	}
	return device, nil
}

func (s *Service) begin(ctx context.Context) (domain.Tx, error) {
	tx, err := s.store.Begin(ctx)
	if err != nil {
		s.log.Error("begin transaction", "code", "INTERNAL")
		return nil, err
	}
	return tx, nil
}
