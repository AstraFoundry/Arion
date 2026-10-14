package account

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// decodeClaims reads a JWT payload for display. It does not verify the
// signature: the sync server is the authority for iss, aud, exp, and ent.
func decodeClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[1] == "" {
		return nil, &Error{Code: CodeEntitlement, Detail: "jwt"}
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(parts[1])
	}
	if err != nil {
		return nil, &Error{Code: CodeEntitlement, Detail: "jwt", err: err}
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, &Error{Code: CodeEntitlement, Detail: "jwt", err: err}
	}
	return claims, nil
}

func entitlements(claims map[string]any) ([]string, error) {
	raw, ok := claims["ent"]
	if !ok || raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, &Error{Code: CodeEntitlement, Detail: "ent_type"}
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		value, ok := item.(string)
		if !ok {
			return nil, &Error{Code: CodeEntitlement, Detail: "ent_type"}
		}
		out = append(out, value)
	}
	return out, nil
}

func stringClaim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return value
}
