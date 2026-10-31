// Package controlsock binds the desktop E2E control socket so only the
// current user can connect, regardless of the process umask.
package controlsock

import (
	"errors"
	"net"
	"os"
)

// Listen binds path and limits it to the current user.
// POSIX mode bits are set to 0600 after bind, which replaces the umask.
// Windows ignores those bits, so the socket file DACL is replaced instead.
func Listen(path string) (net.Listener, error) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := restrictOwner(path); err != nil {
		listener.Close()
		_ = os.Remove(path)
		return nil, err
	}
	return listener, nil
}
