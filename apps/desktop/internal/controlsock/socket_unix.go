//go:build !windows

package controlsock

import "os"

func restrictOwner(path string) error {
	return os.Chmod(path, 0o600)
}
