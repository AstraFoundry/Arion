// Package models holds the reviewed model catalog: provenance, licenses,
// file layout and measured evaluation summaries. It never contains weights.
package models

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
)

//go:embed catalog.json
var catalogJSON []byte

// Catalog is the approved, provisionable model metadata (MOD-01).
type Catalog struct {
	Version   int        `json:"version"`
	Artifacts []Artifact `json:"artifacts"`
	Bundles   []Bundle   `json:"bundles"`
	Rewriters []Rewriter `json:"rewriters"`
}

// Rewriter is an optional local LLM for live rewriting (009). Its weights
// are individual pinned files; the GenAI runtime ships inside the app.
type Rewriter struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Runtime      Runtime               `json:"runtime"`
	Repository   string                `json:"repository"`
	Revision     string                `json:"revision"`
	Source       string                `json:"sourceRepository"`
	ModelCard    string                `json:"modelCard"`
	License      License               `json:"license"`
	Platforms    []string              `json:"platforms"`
	Languages    []string              `json:"languages"`
	Files        map[string]RemoteFile `json:"files"`
	Prompt       string                `json:"prompt"`
	Quantization string                `json:"quantization"`
	Conversion   string                `json:"conversion"`
	Evaluations  []RewriteEvaluation   `json:"evaluations"`
}

// RemoteFile is one individually downloaded, digest-pinned file.
type RemoteFile struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

// RewriteEvaluation is a content-free summary of a live pipeline report.
type RewriteEvaluation struct {
	RunID          string  `json:"runId"`
	Report         string  `json:"report"`
	Hardware       string  `json:"hardware"`
	Split          string  `json:"split"`
	Cases          int     `json:"cases"`
	SourcePass     int     `json:"sourcePass"`
	FinalPass      int     `json:"finalPass"`
	Worsened       int     `json:"worsened"`
	StopToFinalP50 float64 `json:"stopToFinalP50Ms"`
	StopToFinalP95 float64 `json:"stopToFinalP95Ms"`
	PeakRSSBytes   int64   `json:"peakRssBytes,omitempty"`
}

// Rewriter returns a rewriter by ID.
func (c Catalog) Rewriter(id string) (Rewriter, bool) {
	for _, r := range c.Rewriters {
		if r.ID == id {
			return r, true
		}
	}
	return Rewriter{}, false
}

// SizeBytes is a rewriter's total download size.
func (r Rewriter) SizeBytes() int64 {
	var n int64
	for _, f := range r.Files {
		n += f.SizeBytes
	}
	return n
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

// Source pins an archive, or a canonical file inventory for format "files".
// Every payload is size- and digest-verified before loading.
type Source struct {
	Format     string `json:"format,omitempty"`
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
	URL       string `json:"url,omitempty"`
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
	Adopted     bool         `json:"adopted,omitempty"`
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
		if a.Source.Format != "" && a.Source.Format != "files" {
			return Catalog{}, fmt.Errorf("catalog: unsupported source format")
		}
		if a.Source.Format == "files" {
			data, _ := json.Marshal(a.Files)
			digest := sha256.Sum256(data)
			var total int64
			names := map[string]bool{}
			for _, f := range a.Files {
				name := path.Base(f.Member)
				if f.URL == "" || name == "." || name == ".." || name == ".verified.json" || strings.ContainsAny(name, `/\`) || names[name] {
					return Catalog{}, fmt.Errorf("catalog: invalid remote file")
				}
				names[name] = true
				total += f.SizeBytes
			}
			if hex.EncodeToString(digest[:]) != a.Source.SHA256 || total != a.Source.SizeBytes {
				return Catalog{}, fmt.Errorf("catalog: remote file inventory mismatch")
			}
		}
		artifacts[a.ID] = a
	}
	rewriters := map[string]bool{}
	for _, r := range catalog.Rewriters {
		if r.ID == "" || r.Name == "" || rewriters[r.ID] {
			return Catalog{}, fmt.Errorf("catalog: missing or duplicate rewriter id %q", r.ID)
		}
		rewriters[r.ID] = true
		if r.Repository == "" || r.Revision == "" || r.Prompt == "" || r.Runtime.Engine == "" || len(r.Platforms) == 0 {
			return Catalog{}, fmt.Errorf("catalog: %s lacks pinned provenance", r.ID)
		}
		if r.License.Model == "" || r.License.Runtime == "" || r.License.Reference == "" {
			return Catalog{}, fmt.Errorf("catalog: %s lacks license references", r.ID)
		}
		if len(r.Files) == 0 {
			return Catalog{}, fmt.Errorf("catalog: %s has no files", r.ID)
		}
		for name, f := range r.Files {
			if name == "" || strings.ContainsAny(name, `/\`) || f.URL == "" || !sha256Pattern.MatchString(f.SHA256) || f.SizeBytes <= 0 {
				return Catalog{}, fmt.Errorf("catalog: %s file %q lacks digest", r.ID, name)
			}
		}
	}
	bundles := map[string]bool{}
	for _, b := range catalog.Bundles {
		if b.ID == "" || b.Name == "" || bundles[b.ID] {
			return Catalog{}, fmt.Errorf("catalog: missing or duplicate bundle id %q", b.ID)
		}
		bundles[b.ID] = true
		if artifacts[b.Streaming].Purpose != "streaming-asr" && artifacts[b.Streaming].Purpose != "chunked-asr" {
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
