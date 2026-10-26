//go:build windows && (amd64 || arm64)

package mygo

import (
	"errors"
	"fmt"
	win "github.com/egoist/mygo/internal/windows"
	"net"
	"os"
	"syscall"
	"time"
)

// HandleNotificationRegistration handles the installer's exact registration
// command. Call first in main, before bootstrap and RequestSingleInstanceLock;
// return from main when handled. It does not start an app or change OS consent.
func HandleNotificationRegistration() (handled bool, err error) {
	if len(os.Args) != 2 {
		return false, nil
	}
	switch os.Args[1] {
	case "--mygo-register-notifications":
		return true, win.RegisterNotifications(false)
	case "--mygo-unregister-notifications":
		socket := singleInstanceSocket(singleInstanceKey(App.Name()))
		conn, err := net.DialTimeout("unix", socket, time.Second)
		if err == nil {
			conn.Close()
			return true, errors.New("mygo: close the running application before unregistering its notification sender")
		}
		if !errors.Is(err, syscall.ECONNREFUSED) && !errors.Is(err, syscall.ENOENT) && !errors.Is(err, syscall.ERROR_PATH_NOT_FOUND) {
			return true, fmt.Errorf("mygo: cannot confirm notification sender has stopped: %w", err)
		}
		return true, win.RegisterNotifications(true)
	}
	return false, nil
}

func notificationRelayRequested() bool { return len(os.Args) == 2 && os.Args[1] == "-Embedding" }
func relayNotificationActivation(forward func(id, action string) error) error {
	return win.RelayNotificationActivation(forward)
}
