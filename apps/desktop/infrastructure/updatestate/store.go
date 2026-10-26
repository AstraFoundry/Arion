// Package updatestate persists content-free update handles and reminder decisions.
package updatestate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Reminder records submission, never a claim that the OS displayed a banner.
type Reminder struct {
	Key       string    `json:"key"`
	Submitted time.Time `json:"submitted,omitempty"`
	Later     time.Time `json:"later,omitempty"`
	Dismissed bool      `json:"dismissed,omitempty"`
}

type State struct {
	Schema    int        `json:"schema"`
	Stage     string     `json:"stage,omitempty"`
	Version   string     `json:"version,omitempty"`
	Notes     string     `json:"notes,omitempty"`
	Beta      bool       `json:"beta,omitempty"`
	Automatic bool       `json:"automatic,omitempty"`
	Reminders []Reminder `json:"reminders,omitempty"`
}

type Store struct{ Path string }

func (s Store) Load() (State, error) {
	var st State
	f, err := os.Open(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Schema: 1}, nil
	}
	if err != nil {
		return st, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return st, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return st, errors.New("invalid update state file")
	}
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err = d.Decode(&st); err != nil {
		return st, err
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return st, errors.New("trailing update state content")
	}
	if st.Schema != 1 || len(st.Reminders) > 32 {
		return st, errors.New("unsupported update state")
	}
	if (st.Stage == "") != (st.Version == "") {
		return st, errors.New("incomplete staged update")
	}
	for _, r := range st.Reminders {
		if len(r.Key) > 256 {
			return st, errors.New("invalid reminder identity")
		}
	}
	return st, nil
}
func (s Store) Save(st State) error {
	if s.Path == "" {
		return errors.New("update state path missing")
	}
	st.Schema = 1
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return errors.New("update state too large")
	}
	if err = os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.Path), ".updates-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(name, s.Path); err != nil {
		return fmt.Errorf("save update state: %w", err)
	}
	return nil
}
