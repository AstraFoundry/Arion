//go:build darwin

package urlopen

import (
	"errors"
	"os/exec"
)

func openValidatedURL(raw string) error {
	cmd := exec.Command("/usr/bin/open", "--", raw)
	if err := cmd.Run(); err != nil {
		return errors.New("selection url open failed")
	}
	return nil
}
