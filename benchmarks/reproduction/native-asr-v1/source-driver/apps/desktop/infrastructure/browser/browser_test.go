package browser

import "testing"

func TestValidate(t *testing.T) {
	if err := Validate("https://astrafoundry.si/api/auth/oauth2/authorize?x=1"); err != nil {
		t.Fatal(err)
	}
	if err := Validate("http://127.0.0.1:1/callback"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"file:///etc/passwd", "javascript:alert(1)", "https://user:pass@example", ""} {
		if err := Validate(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}
