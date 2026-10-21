//go:build !darwin && !windows

package browser

import "os/exec"

func open(rawURL string) error {
	cmd := exec.Command("xdg-open", rawURL)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
