package logging

import (
	"io"
	"log/slog"
	"strings"
)

// banned attribute keys are never written. Callers must not pass ciphertext
// or credentials; this drops them if they do.
var banned = map[string]struct{}{
	"authorization": {},
	"token":         {},
	"ciphertext":    {},
	"nonce":         {},
	"tag":           {},
	"aad":           {},
	"public_key":    {},
	"password":      {},
	"body":          {},
}

// New returns a JSON logger. level is debug, info, warn, or error.
func New(w io.Writer, level string) (*slog.Logger, error) {
	var parsed slog.Level
	switch strings.ToLower(level) {
	case "debug":
		parsed = slog.LevelDebug
	case "info", "":
		parsed = slog.LevelInfo
	case "warn":
		parsed = slog.LevelWarn
	case "error":
		parsed = slog.LevelError
	default:
		return nil, errLevel
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: parsed,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if _, drop := banned[strings.ToLower(attr.Key)]; drop {
				return slog.Attr{}
			}
			return attr
		},
	})
	return slog.New(handler), nil
}

type levelError struct{}

func (levelError) Error() string { return "invalid log level" }

var errLevel levelError
