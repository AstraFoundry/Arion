//go:build !js

package mygo

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/internal/update"
)

type updateOwnership struct {
	dir    string
	target string
	file   *os.File
	stage  string
}

// The stable lock inode is never removed. The OS releases ownership after a
// crash, while the separate journal pointer still refuses interrupted swaps.
func acquireUpdateOwnership(target string) (*updateOwnership, error) {
	dir := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".update-lock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("update ownership directory is not a directory")
	}
	path := filepath.Join(dir, "owner")
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return nil, errors.New("invalid update ownership file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockUpdateFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("cannot acquire update ownership (%s): %w", dir, err)
	}
	o := &updateOwnership{dir: dir, target: target, file: f}
	marker := filepath.Join(dir, "stage")
	info, err = os.Lstat(marker)
	if errors.Is(err, os.ErrNotExist) {
		return o, nil
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 4096 {
		f.Close()
		return nil, errors.New("invalid update ownership journal")
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		f.Close()
		return nil, err
	}
	o.stage = string(b)
	if o.stage != "" {
		work := filepath.Dir(o.stage)
		if !filepath.IsAbs(o.stage) || filepath.Base(o.stage) != "record.json" || filepath.Dir(work) != filepath.Dir(target) || !strings.HasPrefix(filepath.Base(work), "."+filepath.Base(target)+".update-") {
			f.Close()
			return nil, fmt.Errorf("%w: invalid journal pointer at %s", ErrUpdateRecoveryRequired, marker)
		}
	}
	return o, nil
}

func (o *updateOwnership) close() { o.file.Close() }

func (o *updateOwnership) allow(path string, acknowledging bool) error {
	if o.stage == "" {
		return nil
	}
	for _, path := range []string{filepath.Dir(o.stage), o.stage} {
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("%w: inspect %s: %v", ErrUpdateRecoveryRequired, o.stage, err)
		}
		if path == o.stage && (!info.Mode().IsRegular() || info.Size() >= update.MaxManifestSize) || path != o.stage && !info.IsDir() {
			return fmt.Errorf("%w: invalid journal entry %s", ErrUpdateRecoveryRequired, path)
		}
	}
	f, err := os.Open(o.stage)
	if err != nil {
		return fmt.Errorf("%w: inspect %s: %v", ErrUpdateRecoveryRequired, o.stage, err)
	}
	defer f.Close()
	var record preparedRecord
	if err := json.UnmarshalRead(io.LimitReader(f, update.MaxManifestSize), &record); err != nil {
		return fmt.Errorf("%w: inspect %s: %v", ErrUpdateRecoveryRequired, o.stage, err)
	}
	if record.Format != 1 || record.Target != o.target || record.Manifest.Version == "" {
		return fmt.Errorf("%w: invalid journal identity at %s", ErrUpdateRecoveryRequired, o.stage)
	}
	switch record.Phase {
	case "prepared", "rolled-back", "acknowledged":
		return nil
	case "applied":
		if acknowledging && path == o.stage {
			return nil
		}
	}
	return fmt.Errorf("%w: inspect %s (phase %s)", ErrUpdateRecoveryRequired, o.stage, record.Phase)
}

func (o *updateOwnership) mark(stage string) error {
	f, err := os.OpenFile(filepath.Join(o.dir, "stage.tmp"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = f.WriteString(stage)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(o.dir, "stage.tmp"), filepath.Join(o.dir, "stage")); err != nil {
		return err
	}
	o.stage = stage
	return nil
}
