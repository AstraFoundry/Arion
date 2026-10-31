// Package controlsock binds the desktop E2E control socket so only the
// current user can connect, regardless of the process umask.
package controlsock

import (
	"errors"
	"net"
	"os"
)

// Listen binds path and sets the socket mode to 0600.
func Listen(path string) (net.Listener, error) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		listener.Close()
		_ = os.Remove(path)
		return nil, err
	}
	return listener, nil
}
