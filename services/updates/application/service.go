package application

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"sync"
	"time"

	"arion.local/updates/domain"
)

// ErrNoRelease means the channel has no release for the request.
var ErrNoRelease = errors.New("no release is published on this channel")

// maxIndexSize bounds a channel index read from the bucket.
const maxIndexSize = 1 << 20

// Options configures the service. PresignTTL and CacheTTL must be positive.
type Options struct {
	Bucket     Bucket
	PublicURL  string
	PresignTTL time.Duration
	CacheTTL   time.Duration
	Now        func() time.Time
}

// Service serves the channels the platform publishes. Channel objects are
// cached for CacheTTL, so a publication reaches clients within that time.
type Service struct {
	bucket     Bucket
	publicURL  string
	presignTTL time.Duration
	cacheTTL   time.Duration
	now        func() time.Time

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	data    []byte
	missing bool
	expires time.Time
}

// New returns a Service.
func New(options Options) (*Service, error) {
	if options.Bucket == nil || options.PublicURL == "" || options.PresignTTL <= 0 || options.CacheTTL <= 0 || options.Now == nil {
		return nil, errors.New("updates service options are incomplete")
	}
	return &Service{
		bucket:     options.Bucket,
		publicURL:  options.PublicURL,
		presignTTL: options.PresignTTL,
		cacheTTL:   options.CacheTTL,
		now:        options.Now,
		cache:      map[string]cached{},
	}, nil
}

// Manifest returns the MyGo update manifest of channel for target.
func (s *Service) Manifest(ctx context.Context, channel domain.Channel, target domain.Target) ([]byte, error) {
	data, err := s.read(ctx, domain.ManifestKey(channel, target), domain.MaxManifestSize)
	if err != nil {
		return nil, err
	}
	if err := domain.CheckManifest(data); err != nil {
		return nil, err
	}
	return data, nil
}

// PublicFile is a release file as clients download it from this service.
type PublicFile struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	URL    string `json:"url"`
}

// PublicTarget is the release of a channel for one target.
type PublicTarget struct {
	Version     string     `json:"version"`
	Prerelease  bool       `json:"prerelease"`
	PublishedAt time.Time  `json:"publishedAt"`
	Notes       string     `json:"notes"`
	Installer   PublicFile `json:"installer"`
}

// PublicChannel is what GET /v1/releases/{channel} answers.
type PublicChannel struct {
	Channel   domain.Channel                 `json:"channel"`
	UpdatedAt time.Time                      `json:"updatedAt"`
	Targets   map[domain.Target]PublicTarget `json:"targets"`
}

// Release returns the public summary of channel. A channel the platform
// has not published yet has no targets.
func (s *Service) Release(ctx context.Context, channel domain.Channel) (PublicChannel, error) {
	index, err := s.index(ctx, channel)
	if errors.Is(err, ErrNoRelease) {
		return PublicChannel{Channel: channel, Targets: map[domain.Target]PublicTarget{}}, nil
	}
	if err != nil {
		return PublicChannel{}, err
	}
	out := PublicChannel{Channel: channel, UpdatedAt: index.UpdatedAt, Targets: map[domain.Target]PublicTarget{}}
	for target, release := range index.Targets {
		out.Targets[target] = PublicTarget{
			Version:     release.Version,
			Prerelease:  release.Prerelease,
			PublishedAt: release.PublishedAt,
			Notes:       release.Notes,
			Installer: PublicFile{
				Name:   release.Installer.Name,
				Size:   release.Installer.Size,
				SHA256: release.Installer.SHA256,
				URL:    s.publicURL + "/v1/updates/" + release.Installer.Name,
			},
		}
	}
	return out, nil
}

// InstallerURL returns a short-lived download URL of the installer the
// channel offers for target.
func (s *Service) InstallerURL(ctx context.Context, channel domain.Channel, target domain.Target) (string, error) {
	index, err := s.index(ctx, channel)
	if err != nil {
		return "", err
	}
	release, ok := index.Targets[target]
	if !ok {
		return "", ErrNoRelease
	}
	return s.bucket.PresignGet(domain.FileKey(release.Installer.Name), s.presignTTL, s.now())
}

// FileURL returns a short-lived download URL of a release file that a
// published channel index or manifest names. Staged and withdrawn objects
// stay in the bucket until the platform deletes them, and are not signed
// for download. Whether a named object still exists is answered by the
// bucket at download time.
func (s *Service) FileURL(ctx context.Context, name string) (string, error) {
	name, err := domain.ParseFileName(name)
	if err != nil {
		return "", err
	}
	published, err := s.publishedFile(ctx, name)
	if err != nil {
		return "", err
	}
	if !published {
		return "", ErrNoRelease
	}
	return s.bucket.PresignGet(domain.FileKey(name), s.presignTTL, s.now())
}

func (s *Service) publishedFile(ctx context.Context, name string) (bool, error) {
	for _, channel := range []domain.Channel{domain.Stable, domain.Beta} {
		index, err := s.index(ctx, channel)
		if errors.Is(err, ErrNoRelease) {
			continue
		}
		if err != nil {
			return false, err
		}
		for target, release := range index.Targets {
			if release.Installer.Name == name || release.Update.Name == name {
				return true, nil
			}
			data, err := s.read(ctx, domain.ManifestKey(channel, target), domain.MaxManifestSize)
			if errors.Is(err, ErrNoRelease) {
				continue
			}
			if err != nil {
				return false, err
			}
			if manifestNamesFile(data, name) {
				return true, nil
			}
		}
	}
	return false, nil
}

func manifestNamesFile(data []byte, name string) bool {
	var manifest struct {
		URL    string `json:"url"`
		Deltas []struct {
			URL string `json:"url"`
		} `json:"deltas"`
		Previous []struct {
			URL string `json:"url"`
		} `json:"previous"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	if fileNameFromURL(manifest.URL) == name {
		return true
	}
	for _, item := range manifest.Deltas {
		if fileNameFromURL(item.URL) == name {
			return true
		}
	}
	for _, item := range manifest.Previous {
		if fileNameFromURL(item.URL) == name {
			return true
		}
	}
	return false
}

func fileNameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Path == "" {
		return ""
	}
	name := path.Base(parsed.Path)
	if _, err := domain.ParseFileName(name); err != nil {
		return ""
	}
	return name
}

func (s *Service) index(ctx context.Context, channel domain.Channel) (domain.Index, error) {
	data, err := s.read(ctx, domain.IndexKey(channel), maxIndexSize)
	if err != nil {
		return domain.Index{}, err
	}
	return domain.ParseIndex(data, channel)
}

// read returns a channel object, from the cache while it is fresh. A
// missing object is cached as missing; a failed read is not cached.
func (s *Service) read(ctx context.Context, key string, limit int64) ([]byte, error) {
	now := s.now()
	s.mu.Lock()
	entry, ok := s.cache[key]
	s.mu.Unlock()
	if ok && now.Before(entry.expires) {
		if entry.missing {
			return nil, ErrNoRelease
		}
		return entry.data, nil
	}
	data, err := s.bucket.Get(ctx, key, limit)
	missing := errors.Is(err, ErrNotFound)
	if err != nil && !missing {
		return nil, err
	}
	s.mu.Lock()
	s.cache[key] = cached{data: data, missing: missing, expires: now.Add(s.cacheTTL)}
	s.mu.Unlock()
	if missing {
		return nil, ErrNoRelease
	}
	return data, nil
}
