// Package urlopen launches an already validated selection URL with the
// operating system's URL opener. It does not start a shell.
package urlopen

import "arion.local/core/application"

// Open refuses a scheme that is not https, claude, or codex, then asks the
// OS to open the URL. The error never includes the URL.
func Open(raw string) error {
	if err := application.ValidateSelectionURL(raw); err != nil {
		return err
	}
	return openValidatedURL(raw)
}
