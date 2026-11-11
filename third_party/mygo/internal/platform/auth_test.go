package platform

import "testing"

func TestParseAuthChallenge(t *testing.T) {
	for _, c := range []struct{ challenge, scheme, realm string }{
		{`Basic realm="intranet"`, "basic", "intranet"},
		{`Basic realm="Access to the staging site", charset="UTF-8"`, "basic", "Access to the staging site"},
		{`basic charset="UTF-8", REALM="later"`, "basic", "later"},
		{`Digest realm="http-auth@example.org", qop="auth, auth-int", algorithm=SHA-256, nonce="7ypf/xlj9XXwfDPEoM4URrv/xwf94BcCAzFZH4GiTo0v"`, "digest", "http-auth@example.org"},
		{`Digest qop="auth", realm=bare, nonce="x"`, "digest", "bare"},
		{`Basic realm="say \"hi\" \\ bye"`, "basic", `say "hi" \ bye`},
		{`Basic realm="unterminated`, "basic", "unterminated"},
		{`NTLM`, "ntlm", ""},
		{`Negotiate YIIBhwYGKwYBBQUC`, "negotiate", ""},
		{`  Basic  `, "basic", ""},
		{``, "", ""},
	} {
		scheme, realm := ParseAuthChallenge(c.challenge)
		if scheme != c.scheme || realm != c.realm {
			t.Errorf("ParseAuthChallenge(%q) = %q, %q; want %q, %q", c.challenge, scheme, realm, c.scheme, c.realm)
		}
	}
}
