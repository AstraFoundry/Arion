//go:build !js

package mygo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/egoist/mygo/internal/update"
)

// ErrUpdateRecoveryRequired means an interrupted or failed rollback must be
// inspected before another update can replace the installation. The error
// includes the staging path containing the journal and any preserved backups.
var ErrUpdateRecoveryRequired = errors.New("mygo: update transaction needs recovery")

// PreparedUpdate owns verified bytes next to the installation. Prepare never
// replaces the installation. Keep Path in application state to recover a stage
// after a restart, together with the application's channel/version identity.
// Methods are safe from multiple goroutines. Do not copy a PreparedUpdate.
type PreparedUpdate struct {
	mu       sync.Mutex
	work     string
	key      string
	record   preparedRecord
	consumed bool
	renameFn func(string, string) error
}

type preparedRecord struct {
	Format       int             `json:"format"`
	Phase        string          `json:"phase"`
	Target       string          `json:"target"`
	TargetDigest string          `json:"targetDigest"`
	StageDigest  string          `json:"stageDigest"`
	Manifest     update.Manifest `json:"manifest"`
	Current      string          `json:"current"`
	Delta        bool            `json:"delta"`
	Moves        []preparedMove  `json:"moves,omitempty"`
}

type preparedMove struct {
	Name   string `json:"name"`
	HadOld bool   `json:"hadOld"`
}

// Path returns the persistent record's absolute path, for RecoverPrepared.
func (p *PreparedUpdate) Path() string { return filepath.Join(p.work, "record.json") }

// Version returns the staged release version.
func (p *PreparedUpdate) Version() string { return p.record.Manifest.Version }

// Applied reports that replacement completed, including a completion recovered
// after a crash. The application must clear its persisted stage handle before
// calling Acknowledge; only then may the receipt and old files be deleted.
func (p *PreparedUpdate) Applied() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.record.Phase == "applied" || p.record.Phase == "acknowledged"
}

// Prepare downloads, verifies and extracts an update without changing the app.
// Cancellation discards unfinished work. Apply must be called only after the
// application has blocked new input/resource loads for its restart transaction.
func (up *Update) Prepare(ctx context.Context, progress func(downloaded, total int64)) (*PreparedUpdate, error) {
	target, err := installTarget()
	if err != nil {
		return nil, err
	}
	p, err := prepareUpdate(ctx, up.manifest, up.key, up.current, target, progress)
	if err != nil {
		return nil, fmt.Errorf("mygo: preparing the update: %w", err)
	}
	return p, nil
}

func prepareUpdate(ctx context.Context, m update.Manifest, key, current, target string, progress func(downloaded, total int64)) (_ *PreparedUpdate, result error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pub, err := update.ParsePublicKey(key)
	if err != nil {
		return nil, err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	digest, err := updateTreeDigest(target)
	if err != nil {
		return nil, err
	}
	work, err := os.MkdirTemp(filepath.Dir(target), "."+filepath.Base(target)+".update-")
	if err != nil {
		return nil, fmt.Errorf("cannot write next to the app: %w", err)
	}
	defer func() {
		if result != nil {
			result = errors.Join(result, os.RemoveAll(work))
		}
	}()
	p := &PreparedUpdate{work: work, key: key, record: preparedRecord{Format: 1, Phase: "prepared", Target: target, TargetDigest: digest, Manifest: m, Current: current}}
	unpacked := filepath.Join(work, "app")
	if d := m.Delta(current); d != nil {
		err = applyDelta(ctx, *d, pub, m.Version, target, work, unpacked, progress)
		if err == nil {
			p.record.Delta = true
		} else {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			log.Printf("mygo: downloading the whole update, as its delta update failed: %v", err)
			if err := os.RemoveAll(unpacked); err != nil {
				return nil, err
			}
			if err := os.Remove(filepath.Join(work, "update.delta")); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
	}
	if !p.record.Delta {
		if m.Size <= 0 || m.Size > update.MaxArchiveSize {
			return nil, fmt.Errorf("invalid size %d", m.Size)
		}
		artifact := filepath.Join(work, "update.tar.gz")
		if err := download(ctx, m.URL, m.Size, m.Signature, pub, artifact, progress); err != nil {
			return nil, err
		}
		if err := extractPrepared(p.record, artifact, unpacked); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := updateStageRoot(unpacked); err != nil {
		return nil, err
	}
	p.record.StageDigest, err = updateTreeDigest(unpacked)
	if err != nil {
		return nil, err
	}
	if err := p.save(); err != nil {
		return nil, err
	}
	return p, nil
}

// RecoverPrepared reloads a stage using this build's key and installation
// identity. It verifies the saved signed artifact, reconstructs its expected
// tree and compares both staged bytes and the current installation. Interrupted
// apply transactions are refused; their journal and backups remain available.
func (u *UpdaterModule) RecoverPrepared(path string) (*PreparedUpdate, error) {
	target, err := installTarget()
	if err != nil {
		return nil, err
	}
	return recoverPrepared(path, packageUpdateKey, target)
}

func recoverPrepared(path, key, target string) (*PreparedUpdate, error) {
	target, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	work := filepath.Dir(path)
	if !filepath.IsAbs(path) || filepath.Base(path) != "record.json" || filepath.Dir(work) != filepath.Dir(target) || !strings.HasPrefix(filepath.Base(work), "."+filepath.Base(target)+".update-") {
		return nil, errors.New("invalid update staging path")
	}
	for _, name := range []string{work, path} {
		info, err := os.Lstat(name)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || name == work && !info.IsDir() || name == path && (!info.Mode().IsRegular() || info.Size() >= update.MaxManifestSize) {
			return nil, errors.New("invalid update staging entry")
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var record preparedRecord
	if err := json.UnmarshalRead(io.LimitReader(f, update.MaxManifestSize), &record); err != nil {
		return nil, err
	}
	if record.Format != 1 || record.Target != target || record.Manifest.Version == "" {
		return nil, errors.New("invalid update staging identity")
	}
	p := &PreparedUpdate{work: work, key: key, record: record}
	if record.Phase != "prepared" && record.Phase != "applied" {
		return nil, fmt.Errorf("%w: inspect %s (phase %s)", ErrUpdateRecoveryRequired, path, record.Phase)
	}
	for _, move := range record.Moves {
		if move.Name == "." || move.Name == ".." || filepath.Base(move.Name) != move.Name {
			return nil, errors.New("invalid update journal entry")
		}
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PreparedUpdate) save() error {
	b, err := json.Marshal(p.record)
	if err != nil {
		return err
	}
	if len(b) >= update.MaxManifestSize {
		return errors.New("update staging record is too large")
	}
	path := p.Path() + ".tmp"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	return os.Rename(path, p.Path())
}

func (p *PreparedUpdate) validate() error {
	var digest string
	var err error
	applied := p.record.Phase == "applied"
	if !applied {
		digest, err = updateTreeDigest(p.record.Target)
		if err != nil {
			return err
		}
		if digest != p.record.TargetDigest {
			return errors.New("the installed app changed after update preparation")
		}
		digest, err = updateTreeDigest(filepath.Join(p.work, "app"))
		if err != nil {
			return err
		}
		if digest != p.record.StageDigest {
			return errors.New("the staged update changed after verification")
		}
	}
	pub, err := update.ParsePublicKey(p.key)
	if err != nil {
		return err
	}
	artifact, size, signature := "update.tar.gz", p.record.Manifest.Size, p.record.Manifest.Signature
	if p.record.Delta {
		d := p.record.Manifest.Delta(p.record.Current)
		if d == nil {
			return errors.New("staged delta has no matching manifest")
		}
		artifact, size, signature = "update.delta", d.Size, d.Signature
	}
	if size <= 0 || size > update.MaxArchiveSize {
		return errors.New("invalid staged artifact size")
	}
	f, err := os.Open(filepath.Join(p.work, artifact))
	if err != nil {
		return err
	}
	sum := sha256.New()
	n, err := io.Copy(sum, io.LimitReader(f, size+1))
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if n != size {
		return errors.New("staged artifact size changed")
	}
	if err := update.Verify(pub, sum.Sum(nil), signature); err != nil {
		return err
	}
	// Rebuild from the signed artifact: a record with edited hashes cannot bless
	// modified staged bytes. The saved manifest is metadata, not a trust anchor.
	expected, err := os.MkdirTemp(p.work, "verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(expected)
	record := p.record
	if applied && record.Delta {
		old, err := p.appliedBase(expected)
		if err != nil {
			return err
		}
		record.Target = old
	}
	expected = filepath.Join(expected, "app")
	if err := extractPrepared(record, filepath.Join(p.work, artifact), expected); err != nil {
		return err
	}
	digest, err = updateTreeDigest(expected)
	if err != nil {
		return err
	}
	if digest != p.record.StageDigest {
		return errors.New("staged tree does not match its signed artifact")
	}
	if applied {
		expectedRoot, err := updateStageRoot(expected)
		if err != nil {
			return err
		}
		if err := compareInstalledUpdate(expectedRoot, p.record.Target); err != nil {
			return err
		}
	}
	return nil
}

func extractPrepared(record preparedRecord, artifact, dir string) error {
	f, err := os.Open(artifact)
	if err != nil {
		return err
	}
	defer f.Close()
	if !record.Delta {
		return update.ExtractArchive(f, dir)
	}
	if runtime.GOOS == "darwin" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		dir = filepath.Join(dir, filepath.Base(record.Target))
	}
	d := record.Manifest.Delta(record.Current)
	if d == nil {
		return errors.New("staged delta has no matching manifest")
	}
	return update.ApplyDelta(f, d.Size, d.From, record.Manifest.Version, record.Target, dir)
}

// Discard deletes a prepared stage. An applying or interrupted transaction
// cannot be discarded because it may hold the only recoverable installation.
func (p *PreparedUpdate) Discard() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.consumed {
		return nil
	}
	if p.record.Phase != "prepared" && p.record.Phase != "rolled-back" {
		return fmt.Errorf("%w: inspect %s", ErrUpdateRecoveryRequired, p.Path())
	}
	ownership, err := acquireUpdateOwnership(p.record.Target)
	if err != nil {
		return err
	}
	defer ownership.close()
	if err := ownership.allow(p.Path(), false); err != nil {
		return err
	}
	if phase, err := p.diskPhase(); err != nil {
		return err
	} else if phase != "prepared" && phase != "rolled-back" {
		return fmt.Errorf("%w: stage advanced to %s; inspect %s", ErrUpdateRecoveryRequired, phase, p.Path())
	}
	if ownership.stage == p.Path() {
		if err := ownership.mark(""); err != nil {
			return err
		}
	}
	if err := os.RemoveAll(p.work); err != nil {
		return err
	}
	p.consumed = true
	return nil
}

// Apply revalidates and replaces the app. The application must atomically
// check consent/generation and prevent input and resource loading before this
// call. Once entered, Apply finishes or rolls back; it is not cancellable.
func (p *PreparedUpdate) Apply() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.consumed {
		return errors.New("prepared update was already consumed")
	}
	if p.record.Phase != "prepared" {
		return fmt.Errorf("%w: inspect %s", ErrUpdateRecoveryRequired, p.Path())
	}
	ownership, err := acquireUpdateOwnership(p.record.Target)
	if err != nil {
		return err
	}
	defer ownership.close()
	if err := ownership.allow(p.Path(), false); err != nil {
		return err
	}
	if phase, err := p.diskPhase(); err != nil {
		return err
	} else if phase != "prepared" {
		return fmt.Errorf("%w: stage advanced to %s; inspect %s", ErrUpdateRecoveryRequired, phase, p.Path())
	}
	if err := ownership.mark(p.Path()); err != nil {
		return err
	}
	if err := p.validate(); err != nil {
		return err
	}
	p.record.Phase = "applying"
	if err := p.save(); err != nil {
		p.record.Phase = "prepared"
		return err
	}
	if err := p.swap(); err != nil {
		if rollbackErr := p.rollback(); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("%w: rollback failed; inspect %s: %v", ErrUpdateRecoveryRequired, p.Path(), rollbackErr))
		}
		// Moved staged entries may have been consumed. Do not advertise a retryable
		// stage. The caller can discard it and prepare again after a full rollback.
		p.record.Phase = "rolled-back"
		return errors.Join(err, p.save())
	}
	p.record.Phase = "applied"
	if err := p.save(); err != nil {
		return fmt.Errorf("%w: installed update but cannot record completion at %s: %v", ErrUpdateRecoveryRequired, p.Path(), err)
	}
	// Keep the authenticated source and backups until the application clears
	// its persisted handle and explicitly acknowledges completion. A crash in
	// that interval can then reconcile an already applied update safely.
	if runtime.GOOS == "linux" {
		if err := update.RefreshDesktopEntry(p.record.Target, filepath.Base(startExe)); err != nil {
			log.Printf("mygo: registering the desktop entry of the update: %v", err)
		}
	}
	return nil
}

func updateStageRoot(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" {
		if len(entries) != 1 || !entries[0].IsDir() || !strings.HasSuffix(entries[0].Name(), ".app") {
			return "", errors.New("the update holds no app bundle")
		}
		return filepath.Join(dir, entries[0].Name()), nil
	}
	if len(entries) == 0 {
		return "", errors.New("the update is empty")
	}
	return dir, nil
}

func (p *PreparedUpdate) swap() error {
	src, err := updateStageRoot(filepath.Join(p.work, "app"))
	if err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(p.work, "backup"), 0o700); err != nil {
		return err
	}
	names := []string{filepath.Base(p.record.Target)}
	if runtime.GOOS != "darwin" {
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		names = nil
		for _, e := range entries {
			names = append(names, e.Name())
		}
	}
	for _, name := range names {
		dst, from := p.record.Target, src
		if runtime.GOOS != "darwin" {
			dst, from = filepath.Join(dst, name), filepath.Join(src, name)
		}
		_, statErr := os.Lstat(dst)
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		p.record.Moves = append(p.record.Moves, preparedMove{Name: name, HadOld: statErr == nil})
		// Record intent before either rename so every crash point preserves enough
		// information to restore old entries, including newly added paths.
		if err := p.save(); err != nil {
			p.record.Moves = p.record.Moves[:len(p.record.Moves)-1]
			return err
		}
		if statErr == nil {
			if err := p.rename(dst, filepath.Join(p.work, "backup", name)); err != nil {
				return err
			}
		}
		if err := p.rename(from, dst); err != nil {
			return err
		}
	}
	return nil
}

func (p *PreparedUpdate) rollback() error {
	var failures []error
	for i := len(p.record.Moves) - 1; i >= 0; i-- {
		move := p.record.Moves[i]
		dst := p.record.Target
		if runtime.GOOS != "darwin" {
			dst = filepath.Join(dst, move.Name)
		}
		backup := filepath.Join(p.work, "backup", move.Name)
		if move.HadOld {
			if _, err := os.Lstat(backup); errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				failures = append(failures, err)
				continue
			}
		}
		if err := os.RemoveAll(dst); err != nil {
			failures = append(failures, fmt.Errorf("removing replacement %s: %w", dst, err))
			continue
		}
		if move.HadOld {
			if err := p.rename(backup, dst); err != nil {
				failures = append(failures, fmt.Errorf("restoring %s: %w", dst, err))
			}
		}
	}
	return errors.Join(failures...)
}

// updateTreeDigest hashes names, modes, links and bytes without following links.
func updateTreeDigest(root string) (string, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("update tree root is not a directory")
	}
	sum := sha256.New()
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		size := info.Size()
		if info.IsDir() {
			size = 0
		}
		fmt.Fprintf(sum, "%d:%s:%d:%d:", len(rel), rel, info.Mode(), size)
		switch {
		case info.Mode().IsRegular():
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(sum, f)
			return errors.Join(err, f.Close())
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			fmt.Fprintf(sum, "%d:%s", len(link), link)
		case info.IsDir():
		default:
			return fmt.Errorf("unsupported entry in update tree: %s", path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func (p *PreparedUpdate) rename(from, to string) error {
	if p.renameFn != nil {
		return p.renameFn(from, to)
	}
	return rename(from, to)
}

// Keep the journal until backups are gone: Windows can keep the previous
// executable open until process exit. A later launch retries completed stages.
func cleanupAppliedStage(work string) error {
	if err := os.RemoveAll(filepath.Join(work, "backup")); err != nil {
		return err
	}
	return os.RemoveAll(work)
}

func cleanPreparedUpdateLeftovers() {
	target, err := installTarget()
	if err != nil {
		return
	}
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".update-*", "record.json"))
	if err != nil {
		return
	}
	for _, path := range paths {
		workInfo, workErr := os.Lstat(filepath.Dir(path))
		if workErr != nil || !workInfo.IsDir() {
			continue
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() >= update.MaxManifestSize {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var record preparedRecord
		if json.Unmarshal(b, &record) != nil || record.Format != 1 || record.Phase != "acknowledged" || record.Target != target {
			continue
		}
		ownership, err := acquireUpdateOwnership(target)
		if err != nil {
			log.Printf("mygo: acquiring acknowledged update cleanup: %v", err)
			continue
		}
		cleanupErr := ownership.allow(path, true)
		if cleanupErr == nil && ownership.stage == path {
			cleanupErr = ownership.mark("")
		}
		if cleanupErr == nil {
			cleanupErr = cleanupAppliedStage(filepath.Dir(path))
		}
		ownership.close()
		if cleanupErr != nil {
			log.Printf("mygo: removing acknowledged update stage %s: %v", path, cleanupErr)
		}
	}
}

// Acknowledge releases a completed receipt after the application has durably
// cleared its stage handle. On Windows open old executables defer cleanup to a
// later launch; the acknowledged journal is retained until backups are gone.
func (p *PreparedUpdate) Acknowledge() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.consumed {
		return nil
	}
	if p.record.Phase != "applied" && p.record.Phase != "acknowledged" {
		return errors.New("update replacement has not completed")
	}
	ownership, err := acquireUpdateOwnership(p.record.Target)
	if err != nil {
		return err
	}
	defer ownership.close()
	if err := ownership.allow(p.Path(), true); err != nil {
		return err
	}
	p.record.Phase = "acknowledged"
	if err := p.save(); err != nil {
		p.record.Phase = "applied"
		return err
	}
	if ownership.stage == p.Path() {
		if err := ownership.mark(""); err != nil {
			return err
		}
	}
	p.consumed = true
	if err := cleanupAppliedStage(p.work); err != nil {
		log.Printf("mygo: deferring acknowledged update cleanup %s: %v", p.work, err)
	}
	return nil
}

// appliedBase reconstructs the old delta source from preserved replacements
// and untouched entries. Windows/Linux retain old top-level entries not named
// in the new archive, matching the existing swap contract.
func (p *PreparedUpdate) appliedBase(scratch string) (string, error) {
	backup := filepath.Join(p.work, "backup")
	if runtime.GOOS == "darwin" {
		return filepath.Join(backup, filepath.Base(p.record.Target)), nil
	}
	old := filepath.Join(scratch, "old")
	if err := os.Mkdir(old, 0o755); err != nil {
		return "", err
	}
	moved := map[string]bool{}
	for _, move := range p.record.Moves {
		moved[move.Name] = true
	}
	for _, source := range []string{backup, p.record.Target} {
		entries, err := os.ReadDir(source)
		if err != nil {
			return "", err
		}
		var names []string
		for _, entry := range entries {
			if source == backup || !moved[entry.Name()] {
				names = append(names, entry.Name())
			}
		}
		if len(names) == 0 {
			continue
		}
		archive, err := os.CreateTemp(scratch, "base-*.tar.gz")
		if err != nil {
			return "", err
		}
		err = update.WriteArchive(archive, source, names)
		if err == nil {
			_, err = archive.Seek(0, io.SeekStart)
		}
		if err == nil {
			err = update.ExtractArchive(archive, old)
		}
		err = errors.Join(err, archive.Close())
		if err != nil {
			return "", err
		}
	}
	return old, nil
}

// Compare each signed entry without following links. On Windows/Linux entries
// retained by the established overlay swap remain outside the signed archive;
// macOS bundle replacement is exact and additionally rejects extra entries.
func compareInstalledUpdate(expected, target string) error {
	if runtime.GOOS == "darwin" {
		want, err := updateTreeDigest(expected)
		if err != nil {
			return err
		}
		got, err := updateTreeDigest(target)
		if err != nil {
			return err
		}
		if got != want {
			return errors.New("installed update does not match its signed artifact")
		}
		return nil
	}
	return filepath.WalkDir(expected, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(expected, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		want, err := entry.Info()
		if err != nil {
			return err
		}
		actual := filepath.Join(target, rel)
		got, err := os.Lstat(actual)
		if err != nil {
			return err
		}
		if want.Mode() != got.Mode() {
			return fmt.Errorf("installed update entry mode changed: %s", rel)
		}
		switch {
		case want.IsDir():
			return nil
		case want.Mode()&os.ModeSymlink != 0:
			a, err := os.Readlink(path)
			if err != nil {
				return err
			}
			b, err := os.Readlink(actual)
			if err != nil {
				return err
			}
			if a != b {
				return fmt.Errorf("installed update link changed: %s", rel)
			}
		case want.Mode().IsRegular():
			if want.Size() != got.Size() {
				return fmt.Errorf("installed update entry size changed: %s", rel)
			}
			a, err := updateFileDigest(path)
			if err != nil {
				return err
			}
			b, err := updateFileDigest(actual)
			if err != nil {
				return err
			}
			if a != b {
				return fmt.Errorf("installed update entry changed: %s", rel)
			}
		default:
			return fmt.Errorf("unsupported installed update entry: %s", rel)
		}
		return nil
	})
}

func updateFileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	sum := sha256.New()
	_, err = io.Copy(sum, f)
	err = errors.Join(err, f.Close())
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// A recovered handle may outlive another handle's transaction. Consult the
// journal while owning the target lock before deleting or replacing anything.
func (p *PreparedUpdate) diskPhase() (string, error) {
	f, err := os.Open(p.Path())
	if err != nil {
		return "", err
	}
	defer f.Close()
	var record preparedRecord
	if err := json.UnmarshalRead(io.LimitReader(f, update.MaxManifestSize), &record); err != nil {
		return "", err
	}
	if record.Format != 1 || record.Target != p.record.Target || record.Manifest.Version != p.record.Manifest.Version {
		return "", errors.New("update staging identity changed")
	}
	return record.Phase, nil
}
