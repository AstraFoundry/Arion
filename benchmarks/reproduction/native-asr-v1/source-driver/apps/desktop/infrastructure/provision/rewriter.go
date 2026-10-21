package provision

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"arion.local/core/domain"
	"arion.local/models"
)

// Rewriter files are downloaded individually, verified and published into
// one directory by rename, like archive artifacts.

func (s *Store) rewriterDir(r models.Rewriter) string { return filepath.Join(s.Root, r.ID) }

// InstalledRewriter verifies every file of an installed rewriter.
func (s *Store) InstalledRewriter(r models.Rewriter) (string, error) {
	dir := s.rewriterDir(r)
	data, err := os.ReadFile(filepath.Join(dir, ".verified.json"))
	if err != nil {
		return "", domain.Fail(domain.CodeModelMissing, r.ID)
	}
	var m marker
	if json.Unmarshal(data, &m) != nil || m.Artifact != r.ID {
		return "", domain.Fail(domain.CodeArtifactUnverified, r.ID)
	}
	for name, file := range r.Files {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil || info.Size() != file.SizeBytes || m.Files[name] != file.SHA256 {
			return "", domain.Fail(domain.CodeArtifactUnverified, r.ID+" "+name)
		}
		if digest, err := hashFile(path); err != nil || digest != file.SHA256 {
			return "", domain.Fail(domain.CodeArtifactUnverified, r.ID+" "+name)
		}
	}
	return dir, nil
}

// EnsureRewriter returns a verified installation, downloading missing files.
// Progress reports bytes across all files.
func (s *Store) EnsureRewriter(ctx context.Context, r models.Rewriter, progress Progress) (string, error) {
	total := r.SizeBytes()
	if dir, err := s.InstalledRewriter(r); err == nil {
		progress(r.ID, PhaseReady, total, total)
		return dir, nil
	}
	downloads := filepath.Join(s.Root, ".downloads")
	if err := os.MkdirAll(downloads, 0o700); err != nil {
		return "", domain.Fail(domain.CodeProvisioningFailed, "cache directory")
	}
	var suffix [6]byte
	_, _ = rand.Read(suffix[:])
	staging := filepath.Join(s.Root, ".staging-"+r.ID+"-"+hex.EncodeToString(suffix[:]))
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return "", domain.Fail(domain.CodeProvisioningFailed, "staging")
	}
	defer os.RemoveAll(staging)
	var done int64
	digests := map[string]string{}
	for name, file := range r.Files {
		cached := filepath.Join(downloads, file.SHA256)
		base := done
		err := s.fetch(ctx, r.ID, file.URL, file.SizeBytes, cached, func(id string, phase Phase, n, _ int64) {
			progress(id, phase, base+n, total)
		})
		if err != nil {
			return "", err
		}
		progress(r.ID, PhaseVerifying, done, total)
		if digest, err := hashFile(cached); err != nil || digest != file.SHA256 {
			_ = os.Remove(cached)
			return "", domain.Fail(domain.CodeArtifactUnverified, r.ID+" "+name)
		}
		if err := os.Rename(cached, filepath.Join(staging, name)); err != nil {
			return "", domain.Fail(domain.CodeProvisioningFailed, "staging write")
		}
		digests[name] = file.SHA256
		done += file.SizeBytes
	}
	data, _ := json.Marshal(marker{Artifact: r.ID, Files: digests})
	if err := os.WriteFile(filepath.Join(staging, ".verified.json"), data, 0o600); err != nil {
		return "", domain.Fail(domain.CodeProvisioningFailed, "marker")
	}
	final := s.rewriterDir(r)
	retired := final + ".retired"
	_ = os.RemoveAll(retired)
	if _, err := os.Stat(final); err == nil {
		if err := os.Rename(final, retired); err != nil {
			return "", domain.Fail(domain.CodeProvisioningFailed, "replace")
		}
	}
	if err := os.Rename(staging, final); err != nil {
		_ = os.Rename(retired, final)
		return "", domain.Fail(domain.CodeProvisioningFailed, "publish")
	}
	_ = os.RemoveAll(retired)
	progress(r.ID, PhaseReady, total, total)
	return s.InstalledRewriter(r)
}

// RemoveRewriter evicts a rewriter; its worker must be stopped first.
func (s *Store) RemoveRewriter(r models.Rewriter) error {
	return os.RemoveAll(s.rewriterDir(r))
}
