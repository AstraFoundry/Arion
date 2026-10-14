package domain

import (
	"encoding/base64"
	"encoding/binary"
	"time"
)

// EncodeCursor hides a server revision in an opaque token. The revision is
// server-ordered and is not a logical memory revision.
func EncodeCursor(revision int64) (string, error) {
	if revision < 0 {
		return "", Fail(CodeInvalidEnvelope)
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(revision))
	return base64.RawURLEncoding.EncodeToString(buf[:]), nil
}

// DecodeCursor parses an opaque cursor. A zero revision means the start of
// the log.
func DecodeCursor(cursor string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil || len(raw) != 8 {
		return 0, Fail(CodeInvalidEnvelope)
	}
	value := binary.BigEndian.Uint64(raw)
	if value > uint64(^uint64(0)>>1) {
		return 0, Fail(CodeInvalidEnvelope)
	}
	return int64(value), nil
}

// CursorExpired reports whether a cursor anchored at created is past ttl.
// The caller treats expiry as a full snapshot, not as an empty log.
func CursorExpired(created, now time.Time, ttl time.Duration) bool {
	if created.IsZero() || ttl <= 0 {
		return false
	}
	return !now.Before(created.Add(ttl))
}
