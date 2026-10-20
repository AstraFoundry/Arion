// Package httpapi adapts the updates service to HTTP.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"arion.local/updates/application"
	"arion.local/updates/domain"
)

// Deps wires the HTTP surface.
type Deps struct {
	Service *application.Service
	Log     *slog.Logger
	Now     func() time.Time
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// New returns the handler of /v1/updates, /v1/releases and /v1/downloads.
func New(deps Deps) http.Handler {
	h := &handler{svc: deps.Service, log: deps.Log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /readyz", h.health)
	mux.HandleFunc("GET /v1/updates/{name}", h.update)
	mux.HandleFunc("GET /v1/releases/{channel}", h.release)
	mux.HandleFunc("GET /v1/downloads/{channel}/{target}", h.download)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := deps.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		mux.ServeHTTP(sw, r)
		deps.Log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", deps.Now().Sub(started).Milliseconds(),
		)
	})
}

type handler struct {
	svc *application.Service
	log *slog.Logger
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

// update answers a MyGo manifest request, update-<target>.json?channel=,
// or redirects a release file download to the bucket.
func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if target, ok := domain.ManifestTarget(name); ok {
		channel, err := domain.ParseChannel(r.URL.Query().Get("channel"))
		if err != nil {
			h.fail(w, err)
			return
		}
		data, err := h.svc.Manifest(r.Context(), channel, target)
		if err != nil {
			h.fail(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
		return
	}
	location, err := h.svc.FileURL(name)
	if err != nil {
		h.fail(w, err)
		return
	}
	redirect(w, r, location)
}

func (h *handler) release(w http.ResponseWriter, r *http.Request) {
	channel, err := domain.ParseChannel(r.PathValue("channel"))
	if err != nil || r.PathValue("channel") == "" {
		h.fail(w, domain.ErrInvalidChannel)
		return
	}
	release, err := h.svc.Release(r.Context(), channel)
	if err != nil {
		h.fail(w, err)
		return
	}
	// The landing site reads this summary from the browser.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, release)
}

func (h *handler) download(w http.ResponseWriter, r *http.Request) {
	channel, err := domain.ParseChannel(r.PathValue("channel"))
	if err != nil {
		h.fail(w, err)
		return
	}
	target, err := domain.ParseTarget(r.PathValue("target"))
	if err != nil {
		h.fail(w, err)
		return
	}
	location, err := h.svc.InstallerURL(r.Context(), channel, target)
	if err != nil {
		h.fail(w, err)
		return
	}
	redirect(w, r, location)
}

func redirect(w http.ResponseWriter, r *http.Request, location string) {
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, location, http.StatusFound)
}

func (h *handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidChannel), errors.Is(err, domain.ErrInvalidTarget), errors.Is(err, domain.ErrInvalidFile):
		writeJSON(w, http.StatusBadRequest, apiError{Code: "INVALID_REQUEST", Message: err.Error()})
	case errors.Is(err, application.ErrNoRelease):
		w.Header().Set("Cache-Control", "no-cache")
		writeJSON(w, http.StatusNotFound, apiError{Code: "NO_RELEASE", Message: err.Error()})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusServiceUnavailable, apiError{Code: "CANCELED", Message: "request canceled"})
	default:
		// Invalid published objects and storage failures are server faults:
		// never reported as an empty channel.
		h.log.Error("release storage failed", "code", "STORAGE", "error", err.Error())
		writeJSON(w, http.StatusBadGateway, apiError{Code: "STORAGE", Message: "release storage is unavailable"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
