//go:build !js && !darwin && !linux && !windows

package mygo

import (
	"errors"
	"os"
)

func lockUpdateFile(*os.File) error {
	return errors.New("update ownership is unsupported on this platform")
}
