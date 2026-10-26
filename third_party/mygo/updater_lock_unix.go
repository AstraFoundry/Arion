//go:build darwin || linux

package mygo

import (
	"os"
	"syscall"
)

// The kernel releases this ownership on process exit, including a crash.
func lockUpdateFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
