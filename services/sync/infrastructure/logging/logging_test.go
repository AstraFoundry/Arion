package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestDropsBannedKeys(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(&buf, "info")
	if err != nil {
		t.Fatal(err)
	}
	log.Info("push", "ciphertext", "secret-marker", "device_id", "device-a")
	if strings.Contains(buf.String(), "secret-marker") || strings.Contains(buf.String(), "ciphertext") {
		t.Fatal(buf.String())
	}
	if !strings.Contains(buf.String(), "device-a") {
		t.Fatal(buf.String())
	}
}
