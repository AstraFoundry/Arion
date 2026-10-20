// Package config reads ARION_UPDATES_* process configuration.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config is the process configuration. The secret access key is read from
// the environment or a *_FILE path, never from a repository file.
type Config struct {
	HTTPAddr       string
	PublicURL      string
	S3Endpoint     string
	S3Region       string
	S3Bucket       string
	S3Prefix       string
	S3PathStyle    bool
	S3AccessKeyID  string
	S3SecretKey    string
	S3SessionToken string
	PresignTTL     time.Duration
	CacheTTL       time.Duration
	LogLevel       string
}

// Load reads environ. Every value is explicit: there is no default bucket,
// endpoint or public URL.
func Load(environ []string) (Config, error) {
	env := map[string]string{}
	for _, entry := range environ {
		if key, value, ok := strings.Cut(entry, "="); ok {
			env[key] = value
		}
	}
	cfg := Config{
		HTTPAddr:      value(env, "ARION_UPDATES_HTTP_ADDR", ":8081"),
		PublicURL:     strings.TrimRight(env["ARION_UPDATES_PUBLIC_URL"], "/"),
		S3Endpoint:    env["ARION_UPDATES_S3_ENDPOINT"],
		S3Region:      env["ARION_UPDATES_S3_REGION"],
		S3Bucket:      env["ARION_UPDATES_S3_BUCKET"],
		S3Prefix:      env["ARION_UPDATES_S3_PREFIX"],
		S3AccessKeyID: env["ARION_UPDATES_S3_ACCESS_KEY_ID"],
		LogLevel:      value(env, "ARION_UPDATES_LOG_LEVEL", "info"),
	}
	var err error
	switch raw := value(env, "ARION_UPDATES_S3_PATH_STYLE", "true"); raw {
	case "true", "false":
		cfg.S3PathStyle = raw == "true"
	default:
		return Config{}, fmt.Errorf("ARION_UPDATES_S3_PATH_STYLE must be true or false")
	}
	if cfg.S3SecretKey, err = secret(env, "ARION_UPDATES_S3_SECRET_ACCESS_KEY"); err != nil {
		return Config{}, err
	}
	if cfg.S3SessionToken, err = secret(env, "ARION_UPDATES_S3_SESSION_TOKEN"); err != nil {
		return Config{}, err
	}
	if cfg.PresignTTL, err = duration(env, "ARION_UPDATES_PRESIGN_TTL", 10*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.CacheTTL, err = duration(env, "ARION_UPDATES_CACHE_TTL", 30*time.Second); err != nil {
		return Config{}, err
	}
	return cfg, cfg.validate()
}

func (cfg Config) validate() error {
	public, err := url.Parse(cfg.PublicURL)
	if err != nil || public.Host == "" || public.RawQuery != "" || public.Fragment != "" || public.User != nil ||
		(public.Scheme != "https" && !(public.Scheme == "http" && isLoopback(public.Hostname()))) {
		return fmt.Errorf("ARION_UPDATES_PUBLIC_URL must be an https URL (http only on loopback)")
	}
	for key, value := range map[string]string{
		"ARION_UPDATES_S3_ENDPOINT":          cfg.S3Endpoint,
		"ARION_UPDATES_S3_REGION":            cfg.S3Region,
		"ARION_UPDATES_S3_BUCKET":            cfg.S3Bucket,
		"ARION_UPDATES_S3_ACCESS_KEY_ID":     cfg.S3AccessKeyID,
		"ARION_UPDATES_S3_SECRET_ACCESS_KEY": cfg.S3SecretKey,
	} {
		if value == "" {
			return fmt.Errorf("%s is required", key)
		}
	}
	if cfg.PresignTTL < time.Second || cfg.PresignTTL > 7*24*time.Hour || cfg.CacheTTL <= 0 {
		return fmt.Errorf("ARION_UPDATES_PRESIGN_TTL must be 1s to 168h and ARION_UPDATES_CACHE_TTL positive")
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("ARION_UPDATES_LOG_LEVEL is invalid")
	}
	return nil
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func value(env map[string]string, key, fallback string) string {
	if raw := env[key]; raw != "" {
		return raw
	}
	return fallback
}

func duration(env map[string]string, key string, fallback time.Duration) (time.Duration, error) {
	raw := env[key]
	if raw == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s is invalid", key)
	}
	return parsed, nil
}

func secret(env map[string]string, key string) (string, error) {
	value, file := env[key], env[key+"_FILE"]
	if value != "" && file != "" {
		return "", fmt.Errorf("%s and %s_FILE are both set", key, key)
	}
	if file == "" {
		return value, nil
	}
	body, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("%s_FILE: %w", key, err)
	}
	return strings.TrimSpace(string(body)), nil
}
