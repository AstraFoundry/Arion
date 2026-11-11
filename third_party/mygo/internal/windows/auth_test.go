//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"encoding/pem"
	"strings"
	"testing"
)

func TestPEMCertificate(t *testing.T) {
	der := bytes.Repeat([]byte{0x30, 0x82, 0xfe, 0x01, 0x7f}, 97)
	enc := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	for _, s := range []string{enc, strings.ReplaceAll(enc, "\n", "\r\n")} {
		if got := pemCertificate(s); !bytes.Equal(got, der) {
			t.Errorf("pemCertificate(%q) = %x", s, got)
		}
	}
	for _, s := range []string{"", "-----BEGIN CERTIFICATE-----\n!!!\n-----END CERTIFICATE-----\n", "-----BEGIN CERTIFICATE-----\nMIIB\n"} {
		if got := pemCertificate(s); got != nil {
			t.Errorf("pemCertificate(%q) = %x, want nil", s, got)
		}
	}
}
