package domain

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestDeviceSignature(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"operations":[]}`)
	message := CanonicalRequest("dev", 1_700_000_000_000, "POST", "/v1/sync/push", body)
	signature := ed25519.Sign(private, message)
	if err := VerifySignature(public, message, signature); err != nil {
		t.Fatal(err)
	}
	tampered := CanonicalRequest("dev", 1_700_000_000_000, "POST", "/v1/sync/push", []byte(`{"operations":[{"x":1}]}`))
	if err := VerifySignature(public, tampered, signature); !IsCode(err, CodeUnauthorized) {
		t.Fatalf("tampered err = %v", err)
	}
	encoded := EncodePublicKey(public)
	parsed, err := ParsePublicKey(encoded)
	if err != nil || string(parsed) != string(public) {
		t.Fatalf("parse %v", err)
	}
	if _, err := ParsePublicKey("@@@"); !IsCode(err, CodeInvalidEnvelope) {
		t.Fatal(err)
	}
}

func TestEnrollMessageStable(t *testing.T) {
	message := EnrollMessage("grp", "dev-b", "pub")
	if string(message) != "arion-sync-enroll-v1\ngrp\ndev-b\npub" {
		t.Fatalf("%q", message)
	}
}
