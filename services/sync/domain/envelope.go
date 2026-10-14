package domain

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"time"
)

// Kind is the client-declared record operation. The relay does not interpret
// the ciphertext.
type Kind string

const (
	KindWrite      Kind = "write"
	KindTombstone  Kind = "tombstone"
	KindResolution Kind = "resolution"
)

// Incoming is one push operation after the interface has decoded the wire.
type Incoming struct {
	OperationID      string
	RecordID         string
	SchemaVersion    int
	KeyEpoch         uint64
	Algorithm        string
	Nonce            []byte
	Ciphertext       []byte
	Tag              []byte
	AAD              []byte
	LogicalRevision  uint64
	Kind             Kind
	ExpectedRevision int64
	Parents          []int64
}

// Envelope is a stored ciphertext revision. ScopeID is an opaque partition
// supplied by the caller; this package does not interpret it.
type Envelope struct {
	ServerRevision   int64
	GroupID          string
	ScopeID          string
	RecordID         string
	DeviceID         string
	OperationID      string
	SchemaVersion    int
	KeyEpoch         uint64
	Algorithm        string
	Nonce            []byte
	Ciphertext       []byte
	Tag              []byte
	AAD              []byte
	LogicalRevision  uint64
	Kind             Kind
	ExpectedRevision int64
	Parents          []int64
	Suppressed       bool
	CreatedAt        time.Time
}

// Receipt is the stored result of an accepted operation, returned unchanged
// when the same device retries that operation id.
type Receipt struct {
	DeviceID       string
	OperationID    string
	Fingerprint    []byte
	ServerRevision int64
	Sibling        bool
	Suppressed     bool
	KeyEpoch       uint64
}

// ValidateIncoming checks envelope shape. An unknown schema version and a
// malformed integrity tag are distinct outcomes.
func ValidateIncoming(in Incoming, lim Limits) error {
	if !ValidID(in.OperationID) || !ValidID(in.RecordID) {
		return Fail(CodeInvalidEnvelope)
	}
	if in.SchemaVersion != lim.SchemaVersion {
		return Fail(CodeUnknownSchema)
	}
	if in.KeyEpoch == 0 {
		return Fail(CodeStaleKeyEpoch)
	}
	if !validAlgorithm(in.Algorithm, lim.MaxAlgorithm) {
		return Fail(CodeInvalidEnvelope)
	}
	if len(in.Nonce) < lim.MinNonce || len(in.Nonce) > lim.MaxNonce {
		return Fail(CodeInvalidEnvelope)
	}
	if len(in.Tag) < lim.MinTag || len(in.Tag) > lim.MaxTag {
		return Fail(CodeInvalidTag)
	}
	if len(in.Ciphertext) == 0 || len(in.Ciphertext) > lim.MaxCiphertext {
		return Fail(CodeInvalidEnvelope)
	}
	if len(in.AAD) > lim.MaxAAD {
		return Fail(CodeInvalidEnvelope)
	}
	switch in.Kind {
	case KindWrite, KindTombstone, KindResolution:
	default:
		return Fail(CodeInvalidEnvelope)
	}
	if in.LogicalRevision == 0 || in.ExpectedRevision < 0 {
		return Fail(CodeInvalidEnvelope)
	}
	if len(in.Parents) > lim.MaxParents {
		return Fail(CodeInvalidEnvelope)
	}
	for _, parent := range in.Parents {
		if parent <= 0 {
			return Fail(CodeInvalidEnvelope)
		}
	}
	return nil
}

func validAlgorithm(alg string, max int) bool {
	if len(alg) == 0 || len(alg) > max {
		return false
	}
	for _, r := range alg {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '+' || r == '-':
		default:
			return false
		}
	}
	return true
}

// Fingerprint identifies ciphertext bytes so a second operation id cannot
// replay an envelope the relay already accepted.
func Fingerprint(groupID string, in Incoming) []byte {
	h := sha256.New()
	writeString(h, groupID)
	writeString(h, in.RecordID)
	writeBytes(h, in.Nonce)
	writeBytes(h, in.Ciphertext)
	writeBytes(h, in.Tag)
	return h.Sum(nil)
}

func writeString(h hash.Hash, s string) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(len(s)))
	_, _ = h.Write(buf[:])
	_, _ = h.Write([]byte(s))
}

func writeBytes(h hash.Hash, b []byte) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(len(b)))
	_, _ = h.Write(buf[:])
	_, _ = h.Write(b)
}
