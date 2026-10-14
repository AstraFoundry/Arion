//go:build darwin

package browser

import "os/exec"

func open(rawURL string) error {
	return exec.Command("/usr/bin/open", rawURL).Run()
}
