package macos

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Stage the complete bundle before replacing installed bytes. Keep the old
// bundle until registration/enablement succeeds, and restore it on failure.
func replaceInputBundle(source, destination string, activate, restore func() error) (result error) {
	return replaceInputBundleWithLifecycle(source, destination, activate, restore, inputBundleLifecycle{})
}

type inputBundleLifecycle struct {
	beforeReplace  func() error
	beforeRollback func() error
}

func replaceInputBundleWithLifecycle(source, destination string, activate, restore func() error, lifecycle inputBundleLifecycle) (result error) {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("input method bundle must be a directory")
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	transaction, err := os.MkdirTemp(parent, ".arion-input-install-")
	if err != nil {
		return err
	}
	preserve := false
	defer func() {
		if !preserve {
			result = errors.Join(result, os.RemoveAll(transaction))
		}
	}()
	staged := filepath.Join(transaction, "new.app")
	backup := filepath.Join(transaction, "previous.app")
	if err := copyInputBundle(source, staged); err != nil {
		return err
	}
	if lifecycle.beforeReplace != nil {
		if err := lifecycle.beforeReplace(); err != nil {
			return errors.Join(err, restore())
		}
	}
	handover := lifecycle.beforeReplace != nil
	defer func() {
		if result != nil && handover && !preserve {
			result = errors.Join(result, restore())
		}
	}()
	hadPrevious := false
	if info, err := os.Lstat(destination); err == nil {
		if !info.IsDir() {
			return errors.New("installed input method path must be a directory")
		}
		if err := os.Rename(destination, backup); err != nil {
			return err
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staged, destination); err != nil {
		if hadPrevious {
			if rollback := os.Rename(backup, destination); rollback != nil {
				preserve = true
				return errors.Join(err, fmt.Errorf("input method rollback failed; previous bundle retained at %s: %w", backup, rollback))
			}
		}
		return err
	}
	if err := activate(); err != nil {
		if lifecycle.beforeRollback != nil {
			if stopErr := lifecycle.beforeRollback(); stopErr != nil {
				preserve = true
				return errors.Join(err, fmt.Errorf("input method rollback cannot stop new host; recovery files retained at %s: %w", transaction, stopErr))
			}
		}
		// Retain failed bytes separately while restoring the old path. Never
		// delete the sole previous copy if filesystem restoration fails.
		if rollback := os.Rename(destination, staged); rollback != nil {
			preserve = true
			return errors.Join(err, fmt.Errorf("input method rollback failed; recovery files retained at %s: %w", transaction, rollback))
		}
		if hadPrevious {
			if rollback := os.Rename(backup, destination); rollback != nil {
				preserve = true
				return errors.Join(err, fmt.Errorf("input method rollback failed; previous bundle retained at %s: %w", backup, rollback))
			}
		}
		handover = false
		return errors.Join(err, restore())
	}
	return nil
}

func removeInputBundle(destination string, release func() error) error {
	if err := release(); err != nil {
		return err
	}
	return os.RemoveAll(destination)
}

func copyInputBundle(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, info.Mode().Perm())
		default:
			return fmt.Errorf("unsupported input bundle entry: %s", path)
		}
	})
}
