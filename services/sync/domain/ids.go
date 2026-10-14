package domain

import "regexp"

const maxIDLength = 128

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ValidID reports whether s is an opaque identifier the relay will store.
func ValidID(s string) bool {
	return len(s) >= 1 && len(s) <= maxIDLength && idPattern.MatchString(s)
}
