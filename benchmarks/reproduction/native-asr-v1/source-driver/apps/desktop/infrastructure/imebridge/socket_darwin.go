package imebridge

import (
	"errors"
	"net"
	"os"
	"syscall"
	"time"
)

// Hold an OS-released lock for the listener's lifetime, including stale-path
// recovery. Never remove a live socket or an unrelated filesystem object.
func listenSocket(path string) (net.Listener, func(), error) {
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, nil, err
	}
	release := func() { _ = lock.Close() }
	listener, err := bindSocket(path)
	if err != nil {
		release()
		return nil, nil, err
	}
	return listener, release, nil
}

func bindSocket(path string) (net.Listener, error) {
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, errors.New("input method socket path is not a socket")
		}
		conn, dialErr := net.DialTimeout("unix", path, 200*time.Millisecond)
		if dialErr == nil {
			conn.Close()
			return nil, errors.New("input method socket already has a listener")
		}
		if !errors.Is(dialErr, syscall.ECONNREFUSED) {
			return nil, dialErr
		}
		current, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !os.SameFile(info, current) {
			return nil, errors.New("input method socket changed during recovery")
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return net.Listen("unix", path)
}
