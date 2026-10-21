//go:build !darwin

package netproxy

import "os"

var getenv = os.Getenv

// system reports no proxy: on other platforms only environment variables
// are honored for now.
func system() Settings { return Settings{} }
