package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
)

// -upload refuses to build without the key that signs the updates, which
// a release needs for installed apps to update to it.
func TestCheckUploadNeedsSigningKey(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	_, other, _ := ed25519.GenerateKey(nil)
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIDEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	for _, updates := range []*Updates{
		{PublicKey: base64.StdEncoding.EncodeToString(pub), GitHub: "owner/app"},
		{PublicKey: base64.StdEncoding.EncodeToString(pub), URL: "https://downloads.example.com/app", S3: &S3{Bucket: "app", Region: "us-east-1"}},
	} {
		c := &Config{Updates: updates}
		t.Setenv("MYGO_UPDATER_PRIVATE_KEY", "")
		if err := checkUpload(c); err == nil || !strings.Contains(err.Error(), "MYGO_UPDATER_PRIVATE_KEY") {
			t.Errorf("without the key: %v", err)
		}
		t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(other))
		if err := checkUpload(c); err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Errorf("with another key: %v", err)
		}
		t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(priv))
		if err := checkUpload(c); err != nil {
			t.Errorf("with the key: %v", err)
		}
	}
}
