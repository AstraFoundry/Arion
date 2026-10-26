//go:build !darwin && !windows

package imebridge

import "net"

// The production input-method transport is macOS-only.
func listenSocket(path string) (net.Listener, func(), error) {
	listener, err := net.Listen("unix", path)
	return listener, func() {}, err
}
