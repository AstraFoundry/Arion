package domain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// DeviceStatus is the revocable identity state.
type DeviceStatus string

const (
	DeviceActive  DeviceStatus = "active"
	DeviceRevoked DeviceStatus = "revoked"
)

// Device is an enrolled identity. PublicKey is a raw Ed25519 key. The relay
// uses it to authenticate requests; it is not a group encryption key.
type Device struct {
	ID        string
	GroupID   string
	ScopeID   string
	PublicKey []byte
	Status    DeviceStatus
	CreatedAt time.Time
	RevokedAt *time.Time
}

// ParsePublicKey decodes a raw Ed25519 public key from unpadded base64url.
func ParsePublicKey(encoded string) ([]byte, error) {
	key, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, Fail(CodeInvalidEnvelope)
	}
	return key, nil
}

// EncodePublicKey encodes a raw Ed25519 public key.
func EncodePublicKey(key []byte) string {
	return base64.RawURLEncoding.EncodeToString(key)
}

// CanonicalRequest is the exact byte string a device signs.
//
//	arion-sync-request-v1
//	<device id>
//	<unix milliseconds>
//	<METHOD>
//	<path and raw query>
//	<hex sha256 of the raw body>
func CanonicalRequest(deviceID string, unixMilli int64, method, path string, body []byte) []byte {
	sum := sha256.Sum256(body)
	return []byte(fmt.Sprintf("arion-sync-request-v1\n%s\n%d\n%s\n%s\n%x", deviceID, unixMilli, method, path, sum))
}

// EnrollMessage is the exact byte string a sponsoring device signs to admit
// another device. Recovery-secret enrollment is not represented here.
func EnrollMessage(groupID, deviceID, publicKeyB64 string) []byte {
	return []byte("arion-sync-enroll-v1\n" + groupID + "\n" + deviceID + "\n" + publicKeyB64)
}

// VerifySignature checks an Ed25519 signature over message.
func VerifySignature(publicKey, message, signature []byte) error {
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return Fail(CodeUnauthorized)
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), message, signature) {
		return Fail(CodeUnauthorized)
	}
	return nil
}
