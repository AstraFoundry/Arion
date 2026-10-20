// Command updates serves Arion's update channels and downloads from the
// release bucket that the AstraFoundry platform publishes.
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

	"arion.local/updates/application"
	"arion.local/updates/infrastructure/config"
	"arion.local/updates/infrastructure/s3"
	httpapi "arion.local/updates/interfaces/http"
)

func main() {
	cfg, err := config.Load(os.Environ())
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("configuration rejected", "code", "CONFIG", "error", err.Error())
		os.Exit(1)
	}
	var level slog.Level
	_ = level.UnmarshalText([]byte(cfg.LogLevel))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	bucket, err := s3.New(s3.Config{
		Endpoint:        cfg.S3Endpoint,
		Region:          cfg.S3Region,
		Bucket:          cfg.S3Bucket,
		Prefix:          cfg.S3Prefix,
		PathStyle:       cfg.S3PathStyle,
		AccessKeyID:     cfg.S3AccessKeyID,
		SecretAccessKey: cfg.S3SecretKey,
		SessionToken:    cfg.S3SessionToken,
	}, &http.Client{Timeout: 10 * time.Second}, time.Now)
	if err != nil {
		logger.Error("storage rejected", "code", "CONFIG", "error", err.Error())
		os.Exit(1)
	}
	svc, err := application.New(application.Options{
		Bucket:     bucket,
		PublicURL:  cfg.PublicURL,
		PresignTTL: cfg.PresignTTL,
		CacheTTL:   cfg.CacheTTL,
		Now:        time.Now,
	})
	if err != nil {
		logger.Error("service rejected", "code", "CONFIG")
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(httpapi.Deps{Service: svc, Log: logger, Now: time.Now}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	logger.Info("listening", "addr", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", "code", "SERVER")
		os.Exit(1)
	}
}
