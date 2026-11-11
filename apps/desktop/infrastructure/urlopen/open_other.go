//go:build !darwin && !windows

package urlopen

import "errors"

func openValidatedURL(string) error {
	return errors.New("selection url open is not available")
}
