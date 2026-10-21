// Package provision downloads, verifies and publishes catalog model
// artifacts into the per-user model cache (MOD-02, MOD-03).
package provision

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/netproxy"
	"arion.local/models"
)

// Phase is the user-visible provisioning step.
type Phase string

const (
	PhaseDownloading Phase = "downloading"
	PhaseVerifying   Phase = "verifying"
	PhaseExtracting  Phase = "extracting"
	PhaseReady       Phase = "ready"
)

// Progress reports bytes of the current artifact.
type Progress func(artifact string, phase Phase, done, total int64)

// Store is the verified model cache. Only catalog artifacts are written.
type Store struct {
	Root   string
	Client *http.Client
	// Offline refuses network downloads; installed artifacts still load.
	Offline bool
	// Mirrors maps a download host to hosts serving identical files. When a
	// host is unreachable the download continues from a mirror without
	// asking: catalog digests make the bytes independent of the source.
	Mirrors map[string][]string

	mu        sync.Mutex
	preferred map[string]string
}

// DefaultMirrors serve Hugging Face files where huggingface.co is blocked.
var DefaultMirrors = map[string][]string{"huggingface.co": {"hf-mirror.com"}}

// DefaultRoot is ~/Library/Application Support/Arion/models on macOS and
// %AppData%\Arion\models on Windows.
func DefaultRoot() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Arion", "models"), nil
}

func NewStore(root string) *Store {
	// Bounded dial and handshake times let an unreachable host fail over
	// to a mirror quickly instead of hanging.
	transport := &http.Transport{Proxy: netproxy.Proxy, DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 30 * time.Second, ForceAttemptHTTP2: true}
	return &Store{Root: root, Client: &http.Client{Transport: transport}, Mirrors: DefaultMirrors}
}

// sources orders a URL's host and its mirrors, the host that last worked
// first.
func (s *Store) sources(raw string) []string {
	u, err := url.Parse(raw)
	if err != nil || len(s.Mirrors[u.Host]) == 0 {
		return []string{raw}
	}
	hosts := append([]string{u.Host}, s.Mirrors[u.Host]...)
	s.mu.Lock()
	preferred := s.preferred[u.Host]
	s.mu.Unlock()
	for i, h := range hosts {
		if h == preferred && i > 0 {
			hosts[0], hosts[i] = hosts[i], hosts[0]
		}
	}
	out := make([]string, len(hosts))
	for i, h := range hosts {
		v := *u
		v.Host = h
		out[i] = v.String()
	}
	return out
}

func (s *Store) remember(original, used string) {
	a, errA := url.Parse(original)
	b, errB := url.Parse(used)
	if errA != nil || errB != nil {
		return
	}
	s.mu.Lock()
	if s.preferred == nil {
		s.preferred = map[string]string{}
	}
	s.preferred[a.Host] = b.Host
	s.mu.Unlock()
}

// retryable failures are about reaching a source, not about its bytes.
func retryable(err error) bool {
	coded, ok := err.(*domain.Error)
	return ok && coded.Code == domain.CodeProvisioningFailed && coded.Detail != "offline" && coded.Detail != "write"
}

type marker struct {
	Artifact string            `json:"artifact"`
	Archive  string            `json:"archiveSha256"`
	Files    map[string]string `json:"files"`
}

func (s *Store) dir(a models.Artifact) string { return filepath.Join(s.Root, a.ID) }

// FileName is the on-disk name of an artifact file role.
func FileName(file models.File) string { return path.Base(file.Member) }

// Installed verifies an installed artifact file-by-file. A corrupt or
// partial installation is reported, never loaded (MOD-A01).
func (s *Store) Installed(a models.Artifact) (string, error) {
	if a.Source.Format == "files" {
		return s.InstalledRewriter(remoteArtifact(a))
	}
	dir := s.dir(a)
	data, err := os.ReadFile(filepath.Join(dir, ".verified.json"))
	if err != nil {
		return "", domain.Fail(domain.CodeModelMissing, a.ID)
	}
	var m marker
	if json.Unmarshal(data, &m) != nil || m.Archive != a.Source.SHA256 {
		return "", domain.Fail(domain.CodeArtifactUnverified, a.ID)
	}
	for role, file := range a.Files {
		info, err := os.Stat(filepath.Join(dir, FileName(file)))
		if err != nil || info.Size() != file.SizeBytes {
			return "", domain.Fail(domain.CodeArtifactUnverified, a.ID+" "+role)
		}
		digest, err := hashFile(filepath.Join(dir, FileName(file)))
		if err != nil || digest != file.SHA256 {
			return "", domain.Fail(domain.CodeArtifactUnverified, a.ID+" "+role)
		}
	}
	return dir, nil
}

// Ensure returns a verified installation, downloading it when missing. The
// previous installation stays until a replacement verifies (MOD-A04).
func (s *Store) Ensure(ctx context.Context, a models.Artifact, progress Progress) (string, error) {
	if a.Source.Format == "files" {
		return s.EnsureRewriter(ctx, remoteArtifact(a), progress)
	}
	if dir, err := s.Installed(a); err == nil {
		progress(a.ID, PhaseReady, a.Source.SizeBytes, a.Source.SizeBytes)
		return dir, nil
	}
	if err := os.MkdirAll(filepath.Join(s.Root, ".downloads"), 0o700); err != nil {
		return "", domain.Fail(domain.CodeProvisioningFailed, "cache directory")
	}
	archive := filepath.Join(s.Root, ".downloads", a.Source.SHA256+".tar.bz2")
	if err := s.download(ctx, a, archive, progress); err != nil {
		return "", err
	}
	progress(a.ID, PhaseVerifying, 0, a.Source.SizeBytes)
	digest, err := hashFile(archive)
	if err != nil || digest != a.Source.SHA256 {
		_ = os.Remove(archive)
		return "", domain.Fail(domain.CodeArtifactUnverified, a.ID+" archive digest")
	}
	progress(a.ID, PhaseExtracting, 0, a.Source.SizeBytes)
	if err := s.extract(ctx, a, archive, progress); err != nil {
		return "", err
	}
	_ = os.Remove(archive)
	dir, err := s.Installed(a)
	if err != nil {
		return "", err
	}
	progress(a.ID, PhaseReady, a.Source.SizeBytes, a.Source.SizeBytes)
	return dir, nil
}

func (s *Store) download(ctx context.Context, a models.Artifact, archive string, progress Progress) error {
	return s.fetch(ctx, a.ID, a.Source.URL, a.Source.SizeBytes, archive, progress)
}

// fetch resumes a partial file with a byte range and bounds the size by
// the catalog, so an endpoint cannot fill the disk.
func (s *Store) fetch(ctx context.Context, id, url string, size int64, archive string, progress Progress) error {
	if info, err := os.Stat(archive); err == nil && info.Size() == size {
		return nil
	}
	if s.Offline {
		return domain.Fail(domain.CodeProvisioningFailed, "offline")
	}
	var err error
	for _, source := range s.sources(url) {
		// A partial file resumes from any source: the bytes are identical.
		if err = s.fetchFrom(ctx, id, source, size, archive, progress); err == nil {
			s.remember(url, source)
			return nil
		}
		if !retryable(err) || ctx.Err() != nil {
			return err
		}
	}
	return err
}

func (s *Store) fetchFrom(ctx context.Context, id, url string, size int64, archive string, progress Progress) error {
	partial := archive + ".partial"
	offset := int64(0)
	if info, err := os.Stat(partial); err == nil && info.Size() < size {
		offset = info.Size()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.Fail(domain.CodeProvisioningFailed, "request")
	}
	if offset > 0 {
		request.Header.Set("Range", "bytes="+strconv.FormatInt(offset, 10)+"-")
	}
	response, err := s.Client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return domain.Fail(domain.CodeCancelled, "download cancelled")
		}
		return domain.Fail(domain.CodeProvisioningFailed, "network unavailable")
	}
	defer response.Body.Close()
	flags := os.O_CREATE | os.O_WRONLY
	switch response.StatusCode {
	case http.StatusOK:
		offset = 0
		flags |= os.O_TRUNC
	case http.StatusPartialContent:
		flags |= os.O_APPEND
	default:
		return domain.Fail(domain.CodeProvisioningFailed, "http "+strconv.Itoa(response.StatusCode))
	}
	file, err := os.OpenFile(partial, flags, 0o600)
	if err != nil {
		return domain.Fail(domain.CodeProvisioningFailed, "write")
	}
	done := offset
	buffer := make([]byte, 256*1024)
	last := time.Time{}
	for {
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			if done+int64(n) > size {
				file.Close()
				_ = os.Remove(partial)
				return domain.Fail(domain.CodeArtifactUnverified, "archive larger than catalog size")
			}
			if _, err := file.Write(buffer[:n]); err != nil {
				file.Close()
				return domain.Fail(domain.CodeProvisioningFailed, "write")
			}
			done += int64(n)
			if time.Since(last) > 100*time.Millisecond {
				progress(id, PhaseDownloading, done, size)
				last = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			file.Close()
			if ctx.Err() != nil {
				return domain.Fail(domain.CodeCancelled, "download cancelled")
			}
			return domain.Fail(domain.CodeProvisioningFailed, "connection interrupted")
		}
	}
	if err := file.Close(); err != nil {
		return domain.Fail(domain.CodeProvisioningFailed, "write")
	}
	progress(id, PhaseDownloading, done, size)
	if done != size {
		return domain.Fail(domain.CodeProvisioningFailed, "incomplete download")
	}
	return os.Rename(partial, archive)
}

// extract writes only catalog members into a staging directory, verifies
// them, then publishes atomically by rename.
func (s *Store) extract(ctx context.Context, a models.Artifact, archive string, progress Progress) error {
	wanted := map[string]models.File{}
	for _, file := range a.Files {
		wanted[file.Member] = file
	}
	var suffix [6]byte
	_, _ = rand.Read(suffix[:])
	staging := filepath.Join(s.Root, ".staging-"+a.ID+"-"+hex.EncodeToString(suffix[:]))
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return domain.Fail(domain.CodeProvisioningFailed, "staging")
	}
	defer os.RemoveAll(staging)
	source, err := os.Open(archive)
	if err != nil {
		return domain.Fail(domain.CodeProvisioningFailed, "archive open")
	}
	defer source.Close()
	counter := &countingReader{r: source}
	reader := tar.NewReader(bzip2.NewReader(counter))
	digests := map[string]string{}
	for len(digests) < len(wanted) {
		if ctx.Err() != nil {
			return domain.Fail(domain.CodeCancelled, "extraction cancelled")
		}
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return domain.Fail(domain.CodeArtifactUnverified, "archive structure")
		}
		file, ok := wanted[header.Name]
		if !ok || header.Typeflag != tar.TypeReg || header.Size != file.SizeBytes {
			continue
		}
		out, err := os.OpenFile(filepath.Join(staging, FileName(file)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return domain.Fail(domain.CodeProvisioningFailed, "staging write")
		}
		hash := sha256.New()
		_, copyErr := io.Copy(io.MultiWriter(out, hash), io.LimitReader(reader, file.SizeBytes))
		closeErr := out.Close()
		if copyErr != nil || closeErr != nil {
			return domain.Fail(domain.CodeProvisioningFailed, "staging write")
		}
		digest := hex.EncodeToString(hash.Sum(nil))
		if digest != file.SHA256 {
			return domain.Fail(domain.CodeArtifactUnverified, a.ID+" member digest")
		}
		digests[file.Member] = digest
		progress(a.ID, PhaseExtracting, counter.n, a.Source.SizeBytes)
	}
	if len(digests) != len(wanted) {
		return domain.Fail(domain.CodeArtifactUnverified, a.ID+" members missing")
	}
	data, _ := json.Marshal(marker{Artifact: a.ID, Archive: a.Source.SHA256, Files: digests})
	if err := os.WriteFile(filepath.Join(staging, ".verified.json"), data, 0o600); err != nil {
		return domain.Fail(domain.CodeProvisioningFailed, "marker")
	}
	final := s.dir(a)
	retired := final + ".retired"
	_ = os.RemoveAll(retired)
	if _, err := os.Stat(final); err == nil {
		if err := os.Rename(final, retired); err != nil {
			return domain.Fail(domain.CodeProvisioningFailed, "replace")
		}
	}
	if err := os.Rename(staging, final); err != nil {
		_ = os.Rename(retired, final)
		return domain.Fail(domain.CodeProvisioningFailed, "publish")
	}
	_ = os.RemoveAll(retired)
	return nil
}

// Remove evicts an artifact explicitly. Callers must not remove a model in
// use (MOD-03).
func (s *Store) Remove(a models.Artifact) error {
	if err := os.RemoveAll(s.dir(a)); err != nil {
		return fmt.Errorf("remove %s: %w", a.ID, err)
	}
	return nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func hashFile(name string) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// remoteArtifact shares the verified individual-file provisioning path with LLMs.
func remoteArtifact(a models.Artifact) models.Rewriter {
	files := make(map[string]models.RemoteFile, len(a.Files))
	for _, f := range a.Files {
		files[FileName(f)] = models.RemoteFile{URL: f.URL, SHA256: f.SHA256, SizeBytes: f.SizeBytes}
	}
	return models.Rewriter{ID: a.ID, Files: files}
}
