package diagnostics

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxFiles     = 5
	MaxFileBytes = 2 << 20
	Retention    = 7 * 24 * time.Hour
	queueSize    = 256
)

var ErrClosed = errors.New("diagnostics closed")

type command struct {
	entry    *Entry
	snapshot chan snapshot
}

type snapshot struct {
	files map[string][]byte
	err   error
}

// Recorder has a bounded nonblocking producer queue. One worker owns files;
// exports are barriers in that queue. Dropped records are counted explicitly.
type Recorder struct {
	runID        string
	space        chan struct{}
	draftSession string
	drafted      bool
	dir          string
	commands     chan command
	done         chan struct{}
	mu           sync.Mutex
	closed       bool
	dropped      atomic.Int64
	// Worker-owned state.
	file     *os.File
	size     int64
	opened   time.Time
	failure  error
	maxBytes int64
	now      func() time.Time
}

func Open(dir string) (*Recorder, error) {
	return open(dir, MaxFileBytes, time.Now)
}

func open(dir string, limit int64, now func() time.Time) (*Recorder, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return nil, err
	}
	r := &Recorder{dir: dir, maxBytes: limit, now: now, commands: make(chan command, queueSize), space: make(chan struct{}, 1), done: make(chan struct{})}
	var id [8]byte
	rand.Read(id[:])
	r.runID = hex.EncodeToString(id[:])
	if err := r.prune(); err != nil {
		return nil, err
	}
	if err := r.openFile(); err != nil {
		return nil, err
	}
	go r.work()
	return r, nil
}

func logName(i int) string { return fmt.Sprintf("events.%d.jsonl", i) }

func (r *Recorder) path(i int) string { return filepath.Join(r.dir, logName(i)) }

func (r *Recorder) openFile() error {
	p := r.path(0)
	info, err := os.Lstat(p)
	if err == nil && !info.Mode().IsRegular() {
		return errors.New("diagnostic file is not regular")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	r.file = f
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		r.file = nil
		return err
	}
	r.size, r.opened = stat.Size(), stat.ModTime()
	if r.size == 0 {
		r.opened = r.now()
	}
	return nil
}

func (r *Recorder) prune() error {
	for i := 0; i < MaxFiles; i++ {
		if i == 0 && r.file != nil {
			continue
		}
		p := r.path(i)
		info, err := os.Lstat(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("diagnostic file is not regular")
		}
		if r.now().Sub(info.ModTime()) > Retention {
			if err := os.Remove(p); err != nil {
				return err
			}
		} else if err := os.Chmod(p, 0600); err != nil {
			return err
		}
	}
	return nil
}

func (r *Recorder) rotate() error {
	if err := r.file.Close(); err != nil {
		return err
	}
	r.file = nil
	if err := os.Remove(r.path(MaxFiles - 1)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for i := MaxFiles - 2; i >= 0; i-- {
		if err := os.Rename(r.path(i), r.path(i+1)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return r.openFile()
}

func (r *Recorder) Record(e Entry) {
	if e.InputMethod != nil {
		// Own nested values before validation and asynchronous serialization.
		d := *e.InputMethod
		if d.OSStatus != nil {
			status := *d.OSStatus
			d.OSStatus = &status
		}
		e.InputMethod = &d
	}
	e.At = r.now().UTC()
	e.Run = r.runID
	if !e.valid() || !knownModel(e.ModelID) {
		r.dropped.Add(1)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	select {
	case r.commands <- command{entry: &e}:
	default:
		r.dropped.Add(1)
	}
}

func (r *Recorder) write(e Entry) error {
	if err := r.prune(); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if r.size+int64(len(b)+1) > r.maxBytes || !sameDay(r.opened, r.now()) {
		if err := r.rotate(); err != nil {
			return err
		}
		if err := r.prune(); err != nil {
			return err
		}
	}
	n, err := r.file.Write(append(b, '\n'))
	r.size += int64(n)
	return err
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func (r *Recorder) noteDrops() {
	if n := r.dropped.Swap(0); n > 0 && r.failure == nil {
		r.failure = r.write(Entry{Run: r.runID, At: r.now().UTC(), Event: "events_dropped", Count: int(n)})
	}
}

func (r *Recorder) work() {
	defer close(r.done)
	for cmd := range r.commands {
		select {
		case r.space <- struct{}{}:
		default:
		}
		r.noteDrops()
		if cmd.entry != nil && r.failure == nil {
			r.failure = r.write(*cmd.entry)
		}
		if cmd.snapshot != nil {
			var files map[string][]byte
			if r.failure == nil {
				r.failure = r.file.Sync()
			}
			if r.failure == nil {
				files, r.failure = r.readFiles()
			}
			cmd.snapshot <- snapshot{files: files, err: r.failure}
		}
	}
	r.noteDrops()
	if r.file != nil {
		if err := r.file.Close(); r.failure == nil {
			r.failure = err
		}
	}
}

// Close drains accepted entries. It is safe to call more than once.
func (r *Recorder) Close() error {
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		close(r.commands)
	}
	r.mu.Unlock()
	<-r.done
	return r.failure
}

func (r *Recorder) snapshotFiles() (map[string][]byte, error) {
	reply := make(chan snapshot, 1)
	for {
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			return nil, ErrClosed
		}
		select {
		case r.commands <- command{snapshot: reply}:
			r.mu.Unlock()
			result := <-reply
			return result.files, result.err
		default:
			r.mu.Unlock()
		}
		select {
		case <-r.space:
		case <-r.done:
			return nil, ErrClosed
		}
	}
}
