package center

import (
	"os"
	"testing"
)

func TestCompiledProfile(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if expected := os.Getenv("ARION_TEST_CENTER_URL"); expected != "" && p.URL != expected {
		t.Fatalf("compiled URL = %s", p.URL)
	}
	if expected := os.Getenv("ARION_TEST_CENTER_FINGERPRINT"); expected != "" && p.Fingerprint() != expected {
		t.Fatalf("compiled fingerprint = %s", p.Fingerprint())
	}
	t.Setenv("ARION_CENTER_URL", "https://runtime-override.invalid")
	again, err := Load()
	if err != nil || again.Fingerprint() != p.Fingerprint() {
		t.Fatal("runtime environment changed compiled profile")
	}
}

func TestRejectInvalidProfiles(t *testing.T) {
	for _, raw := range []string{
		`{"url":"http://public.example.org","authMode":"device"}`,
		`{"url":"https://user:password@example.org","authMode":"device"}`,
		`{"url":"https://example.org","authMode":"oidc"}`,
		`{"url":"https://example.org","authMode":"device","issuer":"https://login.example.org"}`,
		`{"url":"https://example.org?secret=x","authMode":"device"}`,
	} {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
