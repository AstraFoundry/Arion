package macos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// Stage the complete bundle before replacing installed bytes. Keep the old
// bundle until registration/enablement succeeds, and restore it on failure.
func replaceInputBundle(source, destination string, activate, restore func() error) (result error) {
	return replaceInputBundleWithLifecycle(source, destination, activate, restore, inputBundleLifecycle{})
}

type inputBundleLifecycle struct {
	beforeReplace   func() error
	beforeRollback  func() error
	validate        func(string) error
	transactionRoot string
}

// Existing bundles are exchanged in one filesystem operation. The other name
// remains the previous complete bundle until activation and commit succeed.
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
	root := lifecycle.transactionRoot
	if root == "" {
		root = filepath.Join(parent, ".arion-transactions")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	pending, err := inspectInputTransactions(root, destination)
	if err != nil {
		return err
	}
	hadPrevious := false
	oldDigest := ""
	if info, err := os.Lstat(destination); err == nil {
		if !info.IsDir() {
			return errors.New("installed input method path must be a directory")
		}
		hadPrevious = true
		oldDigest, err = inputBundleDigest(destination)
		if err != nil {
			return err
		}
		if !inputBundleSwapSupported(parent) {
			return errors.New("input method volume does not support atomic exchange")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	transaction, err := os.MkdirTemp(root, "install-")
	if err != nil {
		return err
	}
	preserve := false
	defer func() {
		if !preserve {
			result = errors.Join(result, os.RemoveAll(transaction))
		}
	}()
	staged := filepath.Join(transaction, "exchange.app")
	if err := copyInputBundle(source, staged); err != nil {
		return err
	}
	if lifecycle.validate != nil {
		if err := lifecycle.validate(staged); err != nil {
			return err
		}
	}
	newDigest, err := inputBundleDigest(staged)
	if err != nil {
		return err
	}
	// Reject cross-volume staging before stopping a usable host.
	var sourceStat, targetStat syscall.Stat_t
	if err := syscall.Stat(staged, &sourceStat); err != nil {
		return err
	}
	if err := syscall.Stat(parent, &targetStat); err != nil {
		return err
	}
	if sourceStat.Dev != targetStat.Dev {
		return errors.New("input method staging and destination are on different volumes")
	}
	record := inputTransaction{Version: 1, Destination: destination, Exchange: staged, Previous: oldDigest, Replacement: newDigest, Phase: "prepared"}
	if err := writeInputTransaction(transaction, record); err != nil {
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
	if hadPrevious {
		current, err := inputBundleDigest(destination)
		if err != nil || current != oldDigest {
			return errors.Join(errors.New("input method changed before exchange"), err)
		}
	}
	if err := exchangeInputBundles(staged, destination, hadPrevious); err != nil {
		return err
	}
	// From this point a crash must retain enough information to inspect both
	// identities, even if the phase write itself fails after the exchange.
	preserve = true
	record.Phase = "exchanged"
	activationErr := writeInputTransaction(transaction, record)
	if activationErr == nil {
		activationErr = activate()
	}
	if activationErr != nil {
		if lifecycle.beforeRollback != nil {
			if stopErr := lifecycle.beforeRollback(); stopErr != nil {
				return errors.Join(activationErr, fmt.Errorf("input method rollback cannot stop new host; recovery files retained at %s: %w", transaction, stopErr))
			}
		}
		if hadPrevious {
			current, checkErr := inputBundleDigest(destination)
			previous, previousErr := inputBundleDigest(staged)
			if checkErr != nil || previousErr != nil || current != newDigest || previous != oldDigest {
				return errors.Join(activationErr, fmt.Errorf("input method rollback identity conflict; recovery files retained at %s", transaction), checkErr, previousErr)
			}
			if err := exchangeInputBundles(staged, destination, true); err != nil {
				return errors.Join(activationErr, fmt.Errorf("input method rollback failed; recovery files retained at %s: %w", transaction, err))
			}
		} else {
			// Registration may already have made this source selectable. A failed
			// first activation is retained for explicit recovery, never deleted here.
			return errors.Join(activationErr, fmt.Errorf("input method first activation incomplete; recovery files retained at %s", transaction))
		}
		handover = false
		record.Phase = "restored"
		if err := writeInputTransaction(transaction, record); err != nil {
			return errors.Join(activationErr, err)
		}
		if err := restore(); err != nil {
			return errors.Join(activationErr, err)
		}
		preserve = false
		return activationErr
	}
	handover = false
	record.Phase = "committed"
	if err := writeInputTransaction(transaction, record); err != nil {
		return err
	}
	// Explicit repair installation has verified the desired build. Only now may
	// older, identity-checked recovery copies be retired.
	for _, directory := range pending {
		if err := os.RemoveAll(directory); err != nil {
			return err
		}
	}
	preserve = false
	return nil
}

type inputTransaction struct {
	Version     int    `json:"version"`
	Destination string `json:"destination"`
	Exchange    string `json:"exchange"`
	Previous    string `json:"previous"`
	Replacement string `json:"replacement"`
	Phase       string `json:"phase"`
}

func writeInputTransaction(directory string, record inputTransaction) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(directory, "transaction.tmp"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(directory, "transaction.tmp"), filepath.Join(directory, "transaction.json")); err != nil {
		return err
	}
	dir, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// Hash ordered relative names, modes, link targets and file contents. This
// identity covers resource-only changes and never follows a bundle symlink.
func inputBundleDigest(bundle string) (string, error) {
	digest := sha256.New()
	err := filepath.Walk(bundle, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(bundle, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(digest, "%d:%s:%d:", len(relative), relative, info.Mode())
		switch {
		case info.IsDir():
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			fmt.Fprint(digest, target)
		case info.Mode().IsRegular():
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			_, readErr := io.Copy(digest, file)
			return errors.Join(readErr, file.Close())
		default:
			return errors.New("unsupported input method bundle entry")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
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

// Interrupted work is never replayed on startup. An explicit install may repair
// an unambiguous transaction, retaining its copies until the new install commits.
func inspectInputTransactions(root, destination string) ([]string, error) {
	pending, err := filepath.Glob(filepath.Join(root, "install-*"))
	if err != nil {
		return nil, err
	}
	current := ""
	if _, err := os.Lstat(destination); err == nil {
		current, err = inputBundleDigest(destination)
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for _, directory := range pending {
		data, err := os.ReadFile(filepath.Join(directory, "transaction.json"))
		if err != nil {
			return nil, fmt.Errorf("input method recovery record unavailable: %w", err)
		}
		var record inputTransaction
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, err
		}
		if record.Version != 1 || record.Destination != destination || record.Exchange != filepath.Join(directory, "exchange.app") ||
			(current != record.Previous && current != record.Replacement) {
			return nil, fmt.Errorf("input method recovery identity conflict: %s", directory)
		}
		if _, err := os.Lstat(record.Exchange); err == nil {
			identity, err := inputBundleDigest(record.Exchange)
			if err != nil || (identity != record.Previous && identity != record.Replacement) {
				return nil, fmt.Errorf("input method recovery copy changed: %s", directory)
			}
		} else if !os.IsNotExist(err) || record.Previous != "" {
			return nil, fmt.Errorf("input method recovery copy missing: %s", directory)
		}
	}
	return pending, nil
}
