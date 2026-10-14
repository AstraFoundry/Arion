//go:build darwin

package keystore

import (
	"errors"
	"os/exec"
	"strings"

	"arion.local/desktop/internal/account"
)

func (Store) Put(token string) error {
	if err := rejectEmpty(token); err != nil {
		return err
	}
	return security(PutArgs(token), false)
}

func (Store) Get() (string, error) {
	out, err := securityOutput(GetArgs())
	if err != nil {
		return "", err
	}
	return strings.TrimRight(out, "\r\n"), nil
}

func (Store) Delete() error {
	return security(DeleteArgs(), true)
}

func security(args []string, missingOK bool) error {
	_, err := securityOutput(args)
	if missingOK && errors.Is(err, account.ErrNotFound) {
		return nil
	}
	return err
}

func securityOutput(args []string) (string, error) {
	cmd := exec.Command("/usr/bin/security", args...)
	out, err := cmd.Output()
	if err == nil {
		return string(out), nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		message := string(exit.Stderr)
		if exit.ExitCode() == 44 || strings.Contains(message, "could not be found") {
			return "", account.ErrNotFound
		}
	}
	return "", account.Wrap(account.CodeStore, "keychain", err)
}
