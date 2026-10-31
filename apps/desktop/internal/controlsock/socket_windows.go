//go:build windows

package controlsock

import (
	"errors"
	"runtime"

	"golang.org/x/sys/windows"
)

// restrictOwner replaces the socket file DACL with one protected allow ACE
// for the current user. os.Chmod only toggles the read-only attribute on
// Windows, and Stat then reports mode 0666, so another user could still
// connect when the path is in a shared directory. A client needs write
// access on the socket file to connect.
func restrictOwner(path string) error {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return err
	}
	sid := user.User.Sid.String()
	if sid == "" {
		return errors.New("current user SID is empty")
	}
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;;GA;;;" + sid + ")")
	if err != nil {
		return err
	}
	dacl, defaulted, err := descriptor.DACL()
	if err != nil {
		return err
	}
	if dacl == nil || defaulted {
		return errors.New("control socket security descriptor has no DACL")
	}
	err = windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	)
	runtime.KeepAlive(descriptor)
	return err
}
