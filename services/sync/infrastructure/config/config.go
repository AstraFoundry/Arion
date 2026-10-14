package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the process configuration. Secrets are read from the environment
// or from a *_FILE path, never from a repository file.
type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	AccountGate   string
	Issuer        string
	Audience      string
	ClientID      string
	Entitlement   string
	JWKSURL       string
	Algorithms    []string
	Leeway        time.Duration
	CacheTTL      time.Duration
	Quota         int64
	MaxBatch      int
	MaxCiphertext int
	MaxBody       int
	CursorTTL     time.Duration
	Skew          time.Duration
	PageSize      int
	LogLevel      string
	RatePerMinute int
}

// Load reads ARION_SYNC_* from environ. Setting both a value and its _FILE
// companion is an error.
func Load(environ []string) (Config, error) {
	env := map[string]string{}
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	cfg := Config{
		HTTPAddr:      value(env, "ARION_SYNC_HTTP_ADDR", ":8080"),
		AccountGate:   value(env, "ARION_SYNC_ACCOUNT_GATE", "noop"),
		Audience:      value(env, "ARION_SYNC_OIDC_AUDIENCE", "https://api.arion.astrafoundry.si"),
		ClientID:      value(env, "ARION_SYNC_OIDC_CLIENT_ID", "arion-sync"),
		Entitlement:   value(env, "ARION_SYNC_OIDC_ENTITLEMENT", "arion.sync.hosted"),
		JWKSURL:       env["ARION_SYNC_OIDC_JWKS_URL"],
		Leeway:        durationOr(env, "ARION_SYNC_OIDC_LEEWAY", 30*time.Second),
		CacheTTL:      durationOr(env, "ARION_SYNC_JWKS_CACHE_TTL", 5*time.Minute),
		Quota:         intOr(env, "ARION_SYNC_DEFAULT_QUOTA_BYTES", 1<<30),
		MaxBatch:      int(intOr(env, "ARION_SYNC_MAX_BATCH", 32)),
		MaxCiphertext: int(intOr(env, "ARION_SYNC_MAX_CIPHERTEXT_BYTES", 256*1024)),
		MaxBody:       int(intOr(env, "ARION_SYNC_MAX_BODY_BYTES", 1<<20)),
		CursorTTL:     durationOr(env, "ARION_SYNC_CURSOR_TTL", 720*time.Hour),
		Skew:          durationOr(env, "ARION_SYNC_AUTH_SKEW", 5*time.Minute),
		PageSize:      int(intOr(env, "ARION_SYNC_PAGE_SIZE", 100)),
		LogLevel:      value(env, "ARION_SYNC_LOG_LEVEL", "info"),
		RatePerMinute: int(intOr(env, "ARION_SYNC_RATE_PER_MINUTE", 120)),
	}
	var err error
	if cfg.DatabaseURL, err = secret(env, "ARION_SYNC_DATABASE_URL"); err != nil {
		return Config{}, err
	}
	if cfg.Issuer, err = secret(env, "ARION_SYNC_OIDC_ISSUER"); err != nil {
		return Config{}, err
	}
	if cfg.Issuer == "" {
		cfg.Issuer = "https://astrafoundry.si"
	}
	algs := value(env, "ARION_SYNC_OIDC_ALGORITHMS", "EdDSA,RS256,ES256")
	for _, alg := range strings.Split(algs, ",") {
		alg = strings.TrimSpace(alg)
		if alg == "" {
			continue
		}
		if strings.EqualFold(alg, "none") || strings.HasPrefix(strings.ToUpper(alg), "HS") {
			return Config{}, fmt.Errorf("ARION_SYNC_OIDC_ALGORITHMS contains a rejected alg")
		}
		cfg.Algorithms = append(cfg.Algorithms, alg)
	}
	if err := cfg.validate(env); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) validate(env map[string]string) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("ARION_SYNC_DATABASE_URL is required")
	}
	if _, err := url.Parse(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("ARION_SYNC_DATABASE_URL is invalid")
	}
	switch cfg.AccountGate {
	case "noop", "oidc":
	default:
		return fmt.Errorf("ARION_SYNC_ACCOUNT_GATE must be noop or oidc")
	}
	if cfg.HTTPAddr == "" {
		return fmt.Errorf("ARION_SYNC_HTTP_ADDR is required")
	}
	if cfg.AccountGate == "oidc" {
		parsed, err := url.Parse(cfg.Issuer)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			return fmt.Errorf("ARION_SYNC_OIDC_ISSUER must be an http or https URL")
		}
		if cfg.Audience == "" || cfg.ClientID == "" || cfg.Entitlement == "" {
			return fmt.Errorf("oidc audience, client id, and entitlement are required")
		}
		if cfg.JWKSURL != "" {
			jwks, err := url.Parse(cfg.JWKSURL)
			if err != nil || (jwks.Scheme != "https" && jwks.Scheme != "http") || jwks.Host == "" {
				return fmt.Errorf("ARION_SYNC_OIDC_JWKS_URL must be an http or https URL")
			}
		}
	}
	if len(cfg.Algorithms) == 0 {
		return fmt.Errorf("ARION_SYNC_OIDC_ALGORITHMS is empty")
	}
	if cfg.Leeway < 0 || cfg.CacheTTL <= 0 || cfg.CursorTTL <= 0 || cfg.Skew <= 0 {
		return fmt.Errorf("sync durations must be positive")
	}
	if cfg.Quota < 0 || cfg.MaxBatch <= 0 || cfg.MaxCiphertext <= 0 || cfg.MaxBody <= 0 || cfg.PageSize <= 0 || cfg.RatePerMinute <= 0 {
		return fmt.Errorf("sync sizes must be positive")
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("ARION_SYNC_LOG_LEVEL is invalid")
	}
	for _, key := range []string{"ARION_SYNC_OIDC_LEEWAY", "ARION_SYNC_JWKS_CACHE_TTL", "ARION_SYNC_CURSOR_TTL", "ARION_SYNC_AUTH_SKEW"} {
		if raw := env[key]; raw != "" {
			if _, err := time.ParseDuration(raw); err != nil {
				return fmt.Errorf("%s is invalid", key)
			}
		}
	}
	return nil
}

func value(env map[string]string, key, fallback string) string {
	if raw, ok := env[key]; ok && raw != "" {
		return raw
	}
	return fallback
}

func durationOr(env map[string]string, key string, fallback time.Duration) time.Duration {
	raw := env[key]
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return -1
	}
	return parsed
}

func intOr(env map[string]string, key string, fallback int64) int64 {
	raw := env[key]
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return -1
	}
	return parsed
}

func secret(env map[string]string, key string) (string, error) {
	value, has := env[key]
	file, hasFile := env[key+"_FILE"]
	if has && value != "" && hasFile && file != "" {
		return "", fmt.Errorf("%s and %s are both set", key, key+"_FILE")
	}
	if hasFile && file != "" {
		body, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("%s: %w", key+"_FILE", err)
		}
		return strings.TrimSpace(string(body)), nil
	}
	return value, nil
}
