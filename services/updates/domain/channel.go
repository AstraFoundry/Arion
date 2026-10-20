// Package domain holds the rules of published update channels: which
// channels and targets exist, which file names may be served, and the
// channel index the AstraFoundry platform publishes.
package domain

import (
	"errors"
	"regexp"
	"strings"
)

// Channel is a published update channel.
type Channel string

const (
	// Stable holds the newest published stable release of each target.
	Stable Channel = "stable"
	// Beta holds the newest published release of each target, stable or
	// prerelease.
	Beta Channel = "beta"
)

// ErrInvalidChannel, ErrInvalidTarget and ErrInvalidFile reject request
// values before any storage key is built from them.
var (
	ErrInvalidChannel = errors.New("unknown update channel")
	ErrInvalidTarget  = errors.New("unknown update target")
	ErrInvalidFile    = errors.New("invalid release file name")
)

// ParseChannel accepts "stable" and "beta"; an empty value is stable, the
// channel of a client that does not ask for one.
func ParseChannel(raw string) (Channel, error) {
	switch Channel(raw) {
	case "", Stable:
		return Stable, nil
	case Beta:
		return Beta, nil
	}
	return "", ErrInvalidChannel
}

// Target is an operating system and architecture, such as darwin-arm64.
type Target string

var targetPattern = regexp.MustCompile(`^(darwin|windows|linux)-(arm64|amd64|universal)$`)

// ParseTarget accepts the targets MyGo builds.
func ParseTarget(raw string) (Target, error) {
	if !targetPattern.MatchString(raw) {
		return "", ErrInvalidTarget
	}
	return Target(raw), nil
}

// ManifestTarget returns the target of an update manifest name,
// update-<target>.json, and whether name is one.
func ManifestTarget(name string) (Target, bool) {
	rest, ok := strings.CutPrefix(name, "update-")
	if !ok {
		return "", false
	}
	rest, ok = strings.CutSuffix(rest, ".json")
	if !ok {
		return "", false
	}
	target, err := ParseTarget(rest)
	return target, err == nil
}

var filePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$`)

// ParseFileName accepts the flat names of release files: installers,
// update archives and deltas. Paths, traversal and manifests are rejected.
func ParseFileName(name string) (string, error) {
	if !filePattern.MatchString(name) || strings.Contains(name, "..") {
		return "", ErrInvalidFile
	}
	if _, manifest := ManifestTarget(name); manifest {
		return "", ErrInvalidFile
	}
	return name, nil
}
