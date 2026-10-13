// Package models holds the reviewed model catalog: provenance, licenses,
// file layout and measured evaluation summaries. It never contains weights.
package models

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
)

//go:embed catalog.json
var catalogJSON []byte

// Catalog is the approved, provisionable model metadata (MOD-01).
type Catalog struct {
	Version   int        `json:"version"`
	Artifacts []Artifact `json:"artifacts"`
	Bundles   []Bundle   `json:"bundles"`
}

// Artifact is one immutable upstream archive.
type Artifact struct {
	ID           string            `json:"id"`
	Purpose      string            `json:"purpose"`
	Kind         string            `json:"kind"`
	Runtime      Runtime           `json:"runtime"`
	Source       Source            `json:"source"`
	License      License           `json:"license"`
	Languages    []string          `json:"languages"`
	Platforms    []string          `json:"platforms"`
	Audio        Audio             `json:"audio"`
	Files        map[string]File   `json:"files"`
	ModelingUnit string            `json:"modelingUnit"`
	LatinCase    string            `json:"latinCase"`
	Punctuation  bool              `json:"punctuation"`
	Quantization string            `json:"quantization"`
	Conversion   string            `json:"conversion"`
	Notes        map[string]string `json:"notes,omitempty"`
}

type Runtime struct {
	Engine  string `json:"engine"`
	Version string `json:"version"`
}

// Source pins the archive: the release asset URL is immutable only together
// with its SHA-256 and byte size, which provisioning verifies.
type Source struct {
	URL        string `json:"url"`
	Release    string `json:"release"`
	Published  string `json:"published"`
	SHA256     string `json:"sha256"`
	SizeBytes  int64  `json:"sizeBytes"`
	ModelCard  string `json:"modelCard"`
	Repository string `json:"repository"`
}

type License struct {
	Model     string `json:"model"`
	Runtime   string `json:"runtime"`
	Reference string `json:"reference"`
	Status    string `json:"status"`
}

type Audio struct {
	SampleRate int `json:"sampleRate"`
	Channels   int `json:"channels"`
}

// File is one archive member the runtime loads; Member is its path inside
// the archive and SHA256 its digest after extraction.
type File struct {
	Member    string `json:"member"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

// Bundle pairs a streaming preview artifact with an optional acoustic
// finalizer. Evaluations are aggregate run summaries.
type Bundle struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Streaming   string       `json:"streaming"`
	Finalizer   string       `json:"finalizer,omitempty"`
	ChunkMs     int          `json:"chunkMs"`
	EndpointMs  int          `json:"endpointSilenceMs"`
	Evaluations []Evaluation `json:"evaluations"`
}

// Evaluation is a content-free summary of a completed benchmark report.
type Evaluation struct {
	RunID             string             `json:"runId"`
	Report            string             `json:"report"`
	Hardware          string             `json:"hardware"`
	Split             string             `json:"split"`
	ErrorRates        map[string]float64 `json:"errorRates"`
	StopToReviewP95Ms float64            `json:"stopToReviewP95Ms"`
	FirstPartialP95Ms float64            `json:"firstPartialP95Ms"`
	RealTimeFactor    float64            `json:"realTimeFactor"`
	Observations      int                `json:"observations"`
}

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Load parses and validates the embedded catalog.
func Load() (Catalog, error) { return Parse(catalogJSON) }

// Parse validates catalog bytes; missing provenance is an error, never a
// default (MOD-A02).
func Parse(data []byte) (Catalog, error) {
	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return Catalog{}, fmt.Errorf("catalog: %w", err)
	}
	if catalog.Version != 1 {
		return Catalog{}, fmt.Errorf("catalog: unsupported version %d", catalog.Version)
	}
	artifacts := map[string]Artifact{}
	for _, a := range catalog.Artifacts {
		if a.ID == "" || artifacts[a.ID].ID != "" {
			return Catalog{}, fmt.Errorf("catalog: missing or duplicate artifact id %q", a.ID)
		}
		if !sha256Pattern.MatchString(a.Source.SHA256) || a.Source.SizeBytes <= 0 || a.Source.URL == "" || a.Source.Release == "" {
			return Catalog{}, fmt.Errorf("catalog: %s lacks pinned source provenance", a.ID)
		}
		if a.License.Model == "" || a.License.Runtime == "" || a.License.Reference == "" {
			return Catalog{}, fmt.Errorf("catalog: %s lacks license references", a.ID)
		}
		if a.Audio.SampleRate <= 0 || a.Audio.Channels != 1 || len(a.Files) == 0 || len(a.Platforms) == 0 || len(a.Languages) == 0 {
			return Catalog{}, fmt.Errorf("catalog: %s lacks runtime metadata", a.ID)
		}
		for role, file := range a.Files {
			if file.Member == "" || !sha256Pattern.MatchString(file.SHA256) || file.SizeBytes <= 0 {
				return Catalog{}, fmt.Errorf("catalog: %s file %s lacks digest", a.ID, role)
			}
		}
		artifacts[a.ID] = a
	}
	bundles := map[string]bool{}
	for _, b := range catalog.Bundles {
		if b.ID == "" || b.Name == "" || bundles[b.ID] {
			return Catalog{}, fmt.Errorf("catalog: missing or duplicate bundle id %q", b.ID)
		}
		bundles[b.ID] = true
		if artifacts[b.Streaming].Purpose != "streaming-asr" {
			return Catalog{}, fmt.Errorf("catalog: bundle %s needs a streaming artifact", b.ID)
		}
		if b.Finalizer != "" && artifacts[b.Finalizer].Purpose != "offline-asr" {
			return Catalog{}, fmt.Errorf("catalog: bundle %s finalizer is not an offline artifact", b.ID)
		}
		if b.ChunkMs <= 0 || b.EndpointMs <= 0 {
			return Catalog{}, fmt.Errorf("catalog: bundle %s lacks pipeline settings", b.ID)
		}
		for _, e := range b.Evaluations {
			if e.RunID == "" || e.Report == "" || e.Hardware == "" || e.Observations <= 0 || len(e.ErrorRates) == 0 {
				return Catalog{}, fmt.Errorf("catalog: bundle %s has an incomplete evaluation", b.ID)
			}
		}
	}
	return catalog, nil
}

// Artifact returns an artifact by ID.
func (c Catalog) Artifact(id string) (Artifact, bool) {
	for _, a := range c.Artifacts {
		if a.ID == id {
			return a, true
		}
	}
	return Artifact{}, false
}

// Bundle returns a bundle by ID.
func (c Catalog) Bundle(id string) (Bundle, bool) {
	for _, b := range c.Bundles {
		if b.ID == id {
			return b, true
		}
	}
	return Bundle{}, false
}

// DownloadBytes is the total archive size a bundle needs.
func (c Catalog) DownloadBytes(b Bundle) int64 {
	var total int64
	for _, id := range []string{b.Streaming, b.Finalizer} {
		if a, ok := c.Artifact(id); ok {
			total += a.Source.SizeBytes
		}
	}
	return total
}
