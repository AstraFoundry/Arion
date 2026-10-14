//go:build windows

package browser

import "os/exec"

func open(rawURL string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
