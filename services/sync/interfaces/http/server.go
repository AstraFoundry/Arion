package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"arion.local/sync/application"
	"arion.local/sync/domain"
)

const (
	headerDevice = "X-Arion-Device-Id"
	headerTime   = "X-Arion-Timestamp"
	headerSig    = "X-Arion-Signature"
)

// Deps wires the HTTP surface. Ready and Now are required.
type Deps struct {
	Service       *application.Service
	Ready         func(context.Context) error
	Log           *slog.Logger
	MaxBody       int
	RatePerMinute int
	Now           func() time.Time
}

// New returns the relay's HTTP handler.
func New(deps Deps) http.Handler {
	server := &handler{
		svc:     deps.Service,
		ready:   deps.Ready,
		log:     deps.Log,
		maxBody: deps.MaxBody,
		now:     deps.Now,
		limit:   &limiter{limit: deps.RatePerMinute, hits: map[string][]time.Time{}},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /readyz", server.readyz)
	mux.HandleFunc("POST /v1/sync/devices", server.register)
	mux.HandleFunc("POST /v1/sync/devices/{id}/revoke", server.revoke)
	mux.HandleFunc("POST /v1/sync/push", server.push)
	mux.HandleFunc("GET /v1/sync/pull", server.pull)
	mux.HandleFunc("GET /v1/sync/snapshot", server.snapshot)
	mux.HandleFunc("POST /v1/sync/ack", server.ack)
	if deps.Service.Mode() == "oidc" {
		mux.HandleFunc("POST /v1/security-events", server.security)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := server.now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		mux.ServeHTTP(sw, r)
		server.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", server.now().Sub(started).Milliseconds(),
			"request_id", requestID(r),
		)
	})
}

type handler struct {
	svc     *application.Service
	ready   func(context.Context) error
	log     *slog.Logger
	maxBody int
	now     func() time.Time
	limit   *limiter
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (h *handler) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if h.ready == nil || h.ready(ctx) != nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Code: "NOT_READY", Message: "not ready"})
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r) {
		return
	}
	body, ok := h.readBody(w, r)
	if !ok {
		return
	}
	var payload struct {
		DeviceID         string `json:"device_id"`
		GroupID          string `json:"group_id"`
		PublicKey        string `json:"public_key"`
		SponsorDeviceID  string `json:"sponsor_device_id"`
		SponsorSignature string `json:"sponsor_signature"`
		Recovery         string `json:"recovery"`
	}
	if !h.decodeJSON(w, body, &payload) {
		return
	}
	proof, ok := h.proof(w, r, body)
	if !ok {
		return
	}
	public, err := domain.ParsePublicKey(payload.PublicKey)
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	var sponsor []byte
	if payload.SponsorSignature != "" {
		sponsor, err = base64.RawURLEncoding.DecodeString(payload.SponsorSignature)
		if err != nil {
			writeError(w, h.log, domain.Fail(domain.CodeUnauthorized))
			return
		}
	}
	out, err := h.svc.Register(r.Context(), application.RegisterInput{
		Authorization:    r.Header.Get("Authorization"),
		Auth:             proof,
		DeviceID:         payload.DeviceID,
		GroupID:          payload.GroupID,
		PublicKey:        public,
		PublicKeyB64:     payload.PublicKey,
		SponsorDeviceID:  payload.SponsorDeviceID,
		SponsorSignature: sponsor,
		Recovery:         payload.Recovery,
	})
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *handler) revoke(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r) {
		return
	}
	body, ok := h.readBody(w, r)
	if !ok {
		return
	}
	if !h.decodeJSON(w, body, &struct{}{}) {
		return
	}
	proof, ok := h.proof(w, r, body)
	if !ok {
		return
	}
	out, err := h.svc.Revoke(r.Context(), application.RevokeInput{
		Authorization: r.Header.Get("Authorization"),
		Auth:          proof,
		TargetID:      r.PathValue("id"),
	})
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) push(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r) {
		return
	}
	body, ok := h.readBody(w, r)
	if !ok {
		return
	}
	var payload struct {
		Operations []operationBody `json:"operations"`
	}
	if !h.decodeJSON(w, body, &payload) {
		return
	}
	ops := make([]domain.Incoming, 0, len(payload.Operations))
	for _, op := range payload.Operations {
		incoming, err := op.decode()
		if err != nil {
			writeError(w, h.log, err)
			return
		}
		ops = append(ops, incoming)
	}
	proof, ok := h.proof(w, r, body)
	if !ok {
		return
	}
	out, err := h.svc.Push(r.Context(), application.PushInput{
		Authorization: r.Header.Get("Authorization"),
		Auth:          proof,
		Operations:    ops,
	})
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) pull(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r) {
		return
	}
	proof, ok := h.proof(w, r, nil)
	if !ok {
		return
	}
	out, err := h.svc.Pull(r.Context(), application.PullInput{
		Authorization: r.Header.Get("Authorization"),
		Auth:          proof,
		Cursor:        r.URL.Query().Get("cursor"),
	})
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, logBody(out))
}

func (h *handler) snapshot(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r) {
		return
	}
	proof, ok := h.proof(w, r, nil)
	if !ok {
		return
	}
	out, err := h.svc.Snapshot(r.Context(), application.SnapshotInput{
		Authorization: r.Header.Get("Authorization"),
		Auth:          proof,
	})
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, logBody(out))
}

func (h *handler) ack(w http.ResponseWriter, r *http.Request) {
	if !h.allow(w, r) {
		return
	}
	body, ok := h.readBody(w, r)
	if !ok {
		return
	}
	var payload struct {
		Revisions []int64 `json:"revisions"`
	}
	if !h.decodeJSON(w, body, &payload) {
		return
	}
	proof, ok := h.proof(w, r, body)
	if !ok {
		return
	}
	out, err := h.svc.Ack(r.Context(), application.AckInput{
		Authorization: r.Header.Get("Authorization"),
		Auth:          proof,
		Revisions:     payload.Revisions,
	})
	if err != nil {
		writeError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) security(w http.ResponseWriter, r *http.Request) {
	media := r.Header.Get("Content-Type")
	if media != "application/secevent+jwt" && !strings.HasPrefix(media, "application/secevent+jwt;") {
		writeJSON(w, http.StatusBadRequest, setError{Err: "invalid_request", Description: "content type rejected"})
		return
	}
	body, ok := h.readBody(w, r)
	if !ok {
		return
	}
	if err := h.svc.ReceiveSecurityEvent(r.Context(), string(body)); err != nil {
		writeError(w, h.log, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *handler) allow(w http.ResponseWriter, r *http.Request) bool {
	id := r.Header.Get(headerDevice)
	if id == "" {
		id = "-"
	}
	if h.limit.allow(id, h.now()) {
		return true
	}
	writeError(w, h.log, domain.Fail(domain.CodeRateLimited))
	return false
}

func (h *handler) proof(w http.ResponseWriter, r *http.Request, body []byte) (application.DeviceProof, bool) {
	millis, err := strconv.ParseInt(r.Header.Get(headerTime), 10, 64)
	if err != nil {
		writeError(w, h.log, domain.Fail(domain.CodeUnauthorized))
		return application.DeviceProof{}, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(r.Header.Get(headerSig))
	if err != nil || len(signature) == 0 {
		writeError(w, h.log, domain.Fail(domain.CodeUnauthorized))
		return application.DeviceProof{}, false
	}
	return application.DeviceProof{
		DeviceID:  r.Header.Get(headerDevice),
		Timestamp: time.UnixMilli(millis).UTC(),
		Signature: signature,
		Method:    r.Method,
		Path:      r.URL.RequestURI(),
		Body:      body,
	}, true
}

type operationBody struct {
	OperationID      string  `json:"operation_id"`
	RecordID         string  `json:"record_id"`
	SchemaVersion    int     `json:"schema_version"`
	KeyEpoch         uint64  `json:"key_epoch"`
	Algorithm        string  `json:"algorithm"`
	Nonce            string  `json:"nonce"`
	Ciphertext       string  `json:"ciphertext"`
	Tag              string  `json:"tag"`
	AAD              string  `json:"aad"`
	LogicalRevision  uint64  `json:"logical_revision"`
	Kind             string  `json:"kind"`
	ExpectedRevision int64   `json:"expected_revision"`
	Parents          []int64 `json:"parents"`
}

func (o operationBody) decode() (domain.Incoming, error) {
	nonce, err := base64.RawURLEncoding.DecodeString(o.Nonce)
	if err != nil {
		return domain.Incoming{}, domain.Fail(domain.CodeInvalidEnvelope)
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(o.Ciphertext)
	if err != nil {
		return domain.Incoming{}, domain.Fail(domain.CodeInvalidEnvelope)
	}
	tag, err := base64.RawURLEncoding.DecodeString(o.Tag)
	if err != nil {
		return domain.Incoming{}, domain.Fail(domain.CodeInvalidTag)
	}
	var aad []byte
	if o.AAD != "" {
		aad, err = base64.RawURLEncoding.DecodeString(o.AAD)
		if err != nil {
			return domain.Incoming{}, domain.Fail(domain.CodeInvalidEnvelope)
		}
	}
	return domain.Incoming{
		OperationID:      o.OperationID,
		RecordID:         o.RecordID,
		SchemaVersion:    o.SchemaVersion,
		KeyEpoch:         o.KeyEpoch,
		Algorithm:        o.Algorithm,
		Nonce:            nonce,
		Ciphertext:       ciphertext,
		Tag:              tag,
		AAD:              aad,
		LogicalRevision:  o.LogicalRevision,
		Kind:             domain.Kind(o.Kind),
		ExpectedRevision: o.ExpectedRevision,
		Parents:          o.Parents,
	}, nil
}

type envelopeDTO struct {
	ServerRevision   int64   `json:"server_revision"`
	GroupID          string  `json:"group_id"`
	RecordID         string  `json:"record_id"`
	DeviceID         string  `json:"device_id"`
	OperationID      string  `json:"operation_id"`
	SchemaVersion    int     `json:"schema_version"`
	KeyEpoch         uint64  `json:"key_epoch"`
	Algorithm        string  `json:"algorithm"`
	Nonce            string  `json:"nonce"`
	Ciphertext       string  `json:"ciphertext"`
	Tag              string  `json:"tag"`
	AAD              string  `json:"aad"`
	LogicalRevision  uint64  `json:"logical_revision"`
	Kind             string  `json:"kind"`
	ExpectedRevision int64   `json:"expected_revision"`
	Parents          []int64 `json:"parents"`
	Suppressed       bool    `json:"suppressed"`
}

type logDTO struct {
	Envelopes []envelopeDTO `json:"envelopes"`
	Cursor    string        `json:"cursor"`
	Snapshot  bool          `json:"snapshot,omitempty"`
	More      bool          `json:"more,omitempty"`
}

func logBody(out application.LogOutput) logDTO {
	dto := logDTO{Cursor: out.Cursor, Snapshot: out.Snapshot, More: out.More, Envelopes: make([]envelopeDTO, 0, len(out.Envelopes))}
	for _, env := range out.Envelopes {
		parents := env.Parents
		if parents == nil {
			parents = []int64{}
		}
		dto.Envelopes = append(dto.Envelopes, envelopeDTO{
			ServerRevision:   env.ServerRevision,
			GroupID:          env.GroupID,
			RecordID:         env.RecordID,
			DeviceID:         env.DeviceID,
			OperationID:      env.OperationID,
			SchemaVersion:    env.SchemaVersion,
			KeyEpoch:         env.KeyEpoch,
			Algorithm:        env.Algorithm,
			Nonce:            base64.RawURLEncoding.EncodeToString(env.Nonce),
			Ciphertext:       base64.RawURLEncoding.EncodeToString(env.Ciphertext),
			Tag:              base64.RawURLEncoding.EncodeToString(env.Tag),
			AAD:              base64.RawURLEncoding.EncodeToString(env.AAD),
			LogicalRevision:  env.LogicalRevision,
			Kind:             string(env.Kind),
			ExpectedRevision: env.ExpectedRevision,
			Parents:          parents,
			Suppressed:       env.Suppressed,
		})
	}
	return dto
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Resync  string `json:"resync,omitempty"`
}

type setError struct {
	Err         string `json:"err"`
	Description string `json:"description"`
}

func writeError(w http.ResponseWriter, log *slog.Logger, err error) {
	if set, ok := application.AsSET(err); ok {
		log.Info("security event rejected", "err", set.Code)
		writeJSON(w, http.StatusBadRequest, setError{Err: set.Code, Description: set.Description})
		return
	}
	var de *domain.Error
	if errors.As(err, &de) {
		body := apiError{Code: de.Code, Message: de.Message}
		if de.Code == domain.CodeCursorExpired {
			body.Resync = "/v1/sync/snapshot"
		}
		log.Info("request rejected", "code", de.Code)
		writeJSON(w, statusFor(de.Code), body)
		return
	}
	var ae *application.Error
	if errors.As(err, &ae) {
		log.Info("request rejected", "code", ae.Code)
		writeJSON(w, statusFor(ae.Code), apiError{Code: ae.Code, Message: ae.Message})
		return
	}
	log.Error("request failed", "code", "INTERNAL")
	writeJSON(w, http.StatusInternalServerError, apiError{Code: "INTERNAL", Message: "internal error"})
}

func statusFor(code string) int {
	switch code {
	case domain.CodeInvalidEnvelope, domain.CodeUnknownSchema, domain.CodeInvalidTag, domain.CodeRecoveryUnsupported:
		return http.StatusBadRequest
	case domain.CodeUnauthorized, application.CodeSessionRevoked:
		return http.StatusUnauthorized
	case domain.CodeDeviceRevoked, domain.CodeScopeMismatch, application.CodeEntitlementRequired, application.CodeTenantFrozen, application.CodeQuotaExceeded:
		return http.StatusForbidden
	case domain.CodeNotFound:
		return http.StatusNotFound
	case domain.CodeConflict, domain.CodeStaleKeyEpoch, domain.CodeReplay, domain.CodeOperationMismatch:
		return http.StatusConflict
	case domain.CodeCursorExpired:
		return http.StatusGone
	case domain.CodePayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case domain.CodeRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func (h *handler) readBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, int64(h.maxBody)+1))
	if err != nil {
		writeError(w, h.log, domain.Fail(domain.CodeInvalidEnvelope))
		return nil, false
	}
	if len(body) > h.maxBody {
		writeError(w, h.log, domain.Fail(domain.CodePayloadTooLarge))
		return nil, false
	}
	return body, true
}

func (h *handler) decodeJSON(w http.ResponseWriter, body []byte, dest any) bool {
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil || dec.More() {
		writeError(w, h.log, domain.Fail(domain.CodeInvalidEnvelope))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func requestID(r *http.Request) string {
	id := r.Header.Get("X-Request-Id")
	if domain.ValidID(id) {
		return id
	}
	return "-"
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

type limiter struct {
	mu    sync.Mutex
	hits  map[string][]time.Time
	limit int
}

func (l *limiter) allow(id string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := now.Add(-time.Minute)
	kept := l.hits[id][:0]
	for _, ts := range l.hits[id] {
		if ts.After(cut) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= l.limit {
		l.hits[id] = kept
		return false
	}
	l.hits[id] = append(kept, now)
	return true
}
