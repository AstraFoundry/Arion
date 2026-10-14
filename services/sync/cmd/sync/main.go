package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"arion.local/sync/application"
	"arion.local/sync/domain"
	"arion.local/sync/infrastructure/config"
	"arion.local/sync/infrastructure/logging"
	"arion.local/sync/infrastructure/noop"
	"arion.local/sync/infrastructure/oidc"
	"arion.local/sync/infrastructure/postgres"
	httpapi "arion.local/sync/interfaces/http"
)

func main() {
	cfg, err := config.Load(os.Environ())
	if err != nil {
		logger, logErr := logging.New(os.Stderr, "error")
		if logErr != nil {
			os.Exit(1)
		}
		logger.Error("configuration rejected", "code", "CONFIG")
		os.Exit(1)
	}
	logger, err := logging.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		os.Exit(1)
	}
	if len(os.Args) != 2 || (os.Args[1] != "serve" && os.Args[1] != "migrate") {
		logger.Error("usage", "code", "USAGE")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database rejected", "code", "DATABASE")
		os.Exit(1)
	}
	defer pool.Close()
	if os.Args[1] == "migrate" {
		if err := postgres.Migrate(ctx, pool); err != nil {
			logger.Error("migration failed", "code", "MIGRATION")
			os.Exit(1)
		}
		logger.Info("migrations applied")
		return
	}
	store := postgres.New(pool)
	var gate application.Gate
	var events application.EventVerifier
	switch cfg.AccountGate {
	case "noop":
		gate = noop.New(cfg.Quota)
	case "oidc":
		provider, err := oidc.New(oidc.Options{
			Issuer:      cfg.Issuer,
			Audience:    cfg.Audience,
			ClientID:    cfg.ClientID,
			Entitlement: cfg.Entitlement,
			JWKSURL:     cfg.JWKSURL,
			Algorithms:  cfg.Algorithms,
			Leeway:      cfg.Leeway,
			CacheTTL:    cfg.CacheTTL,
			Quota:       cfg.Quota,
			Directory:   store,
			Now:         time.Now,
		})
		if err != nil {
			logger.Error("oidc rejected", "code", "CONFIG")
			os.Exit(1)
		}
		gate = provider
		events = provider
	default:
		logger.Error("configuration rejected", "code", "CONFIG")
		os.Exit(1)
	}
	limits := domain.DefaultLimits()
	limits.MaxBatch = cfg.MaxBatch
	limits.MaxCiphertext = cfg.MaxCiphertext
	svc, err := application.New(application.Options{
		Store:     store,
		Directory: store,
		Accounts:  store,
		Gate:      gate,
		Events:    events,
		Limits:    limits,
		CursorTTL: cfg.CursorTTL,
		Skew:      cfg.Skew,
		PageSize:  cfg.PageSize,
		Now:       time.Now,
		Log:       logger,
	})
	if err != nil {
		logger.Error("service rejected", "code", "CONFIG")
		os.Exit(1)
	}
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(httpapi.Deps{Service: svc, Ready: store.Ping, Log: logger, MaxBody: cfg.MaxBody, RatePerMinute: cfg.RatePerMinute, Now: time.Now}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdown)
	}()
	logger.Info("listening", "addr", cfg.HTTPAddr, "gate", cfg.AccountGate)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", "code", "SERVER")
		os.Exit(1)
	}
}

// slog.Logger satisfies application.Logger.
var _ application.Logger = (*slog.Logger)(nil)
