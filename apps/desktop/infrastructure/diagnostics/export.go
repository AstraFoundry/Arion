package diagnostics

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Manifest describes the format and exclusions, never the profile or device.
type Manifest struct {
	Schema        int       `json:"schema"`
	Version       string    `json:"version,omitempty"`
	OS            string    `json:"os"`
	Arch          string    `json:"arch"`
	Exported      time.Time `json:"exported"`
	MaxFiles      int       `json:"maxFiles"`
	MaxFileBytes  int       `json:"maxFileBytes"`
	RetentionDays int       `json:"retentionDays"`
	Excluded      []string  `json:"excluded"`
}

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][a-zA-Z0-9.-]+)?$`)

func (r *Recorder) readFiles() (map[string][]byte, error) {
	files := map[string][]byte{}
	for i := 0; i < MaxFiles; i++ {
		p := r.path(i)
		info, err := os.Lstat(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Size() > MaxFileBytes {
			return nil, errors.New("invalid diagnostic file")
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
		closeErr := f.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(data) > MaxFileBytes {
			return nil, errors.New("diagnostic file exceeds limit")
		}
		var clean bytes.Buffer
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			var e Entry
			dec := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&e); err != nil {
				return nil, err
			}
			if !e.valid() {
				return nil, errInvalidEntry
			}
			if err := dec.Decode(new(any)); err != io.EOF {
				return nil, errInvalidEntry
			}
			if r.now().Sub(e.At) > Retention {
				continue
			}
			if err := json.NewEncoder(&clean).Encode(e); err != nil {
				return nil, err
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		files[filepath.Base(p)] = clean.Bytes()
	}
	return files, nil
}

// Export flushes the queue, snapshots only owned log files, revalidates their
// schema, and atomically publishes a private ZIP at the user's chosen path.
// It never walks the profile directory or includes native stderr/crash dumps.
func (r *Recorder) Export(path, version string) error {
	if !strings.EqualFold(filepath.Ext(path), ".zip") {
		return errors.New("diagnostic export requires zip extension")
	}
	if version != "" && !versionPattern.MatchString(version) {
		return errors.New("invalid diagnostic version")
	}
	if err := r.validDestination(path); err != nil {
		return err
	}
	files, err := r.snapshotFiles()
	if err != nil {
		return err
	}
	manifest := Manifest{Schema: 1, Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Exported: r.now().UTC(),
		MaxFiles: MaxFiles, MaxFileBytes: MaxFileBytes, RetentionDays: 7,
		Excluded: []string{"audio", "transcripts", "prompts", "context", "vocabulary", "preferences", "usage", "paths", "credentials", "native_stderr"}}
	metadata, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	files["manifest.json"] = metadata
	tmp, err := os.CreateTemp(filepath.Dir(path), ".arion-diagnostics-*.zip")
	if err != nil {
		return err
	}
	defer func() {
		if tmp != nil {
			tmp.Close()
		}
	}()
	// Keep the temporary name separately so cleanup also works after Close.
	name := tmp.Name()
	defer os.Remove(name)
	archive := zip.NewWriter(tmp)
	for _, filename := range append([]string{"manifest.json"}, logNames()...) {
		data, ok := files[filename]
		if !ok {
			continue
		}
		h := &zip.FileHeader{Name: filename, Method: zip.Deflate}
		h.SetMode(0600)
		entry, createErr := archive.CreateHeader(h)
		if createErr != nil {
			return createErr
		}
		if _, writeErr := entry.Write(data); writeErr != nil {
			return writeErr
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	tmp = nil
	return os.Rename(name, path)
}

func logNames() []string {
	names := make([]string, MaxFiles)
	for i := 0; i < MaxFiles; i++ {
		names[i] = logName(MaxFiles - 1 - i)
	}
	return names
}

func (r *Recorder) validDestination(path string) error {
	// Never replace a log source or any file inside its directory.
	dir, err := filepath.EvalSymlinks(r.dir)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(dir, filepath.Join(parent, filepath.Base(path)))
	if err != nil {
		return err
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return errors.New("export destination is inside logs")
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("export destination is not regular")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
