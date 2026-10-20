package domain

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

// IndexSchema identifies the channel index format.
const IndexSchema = "astrafoundry.release-channel.v1"

// ErrInvalidIndex means a stored channel index does not follow the contract.
var ErrInvalidIndex = errors.New("channel index is invalid")

// File is a published release file.
type File struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// TargetRelease is the release a channel offers for one target.
type TargetRelease struct {
	Version     string    `json:"version"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"publishedAt"`
	Notes       string    `json:"notes"`
	Installer   File      `json:"installer"`
	Update      File      `json:"update"`
}

// Index is channels/<channel>/index.json, written by the platform when a
// release is published or withdrawn.
type Index struct {
	Schema    string                   `json:"schema"`
	Product   string                   `json:"product"`
	Channel   Channel                  `json:"channel"`
	UpdatedAt time.Time                `json:"updatedAt"`
	Targets   map[Target]TargetRelease `json:"targets"`
}

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ParseIndex decodes and validates an index for channel.
func ParseIndex(data []byte, channel Channel) (Index, error) {
	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return Index{}, ErrInvalidIndex
	}
	if index.Schema != IndexSchema || index.Channel != channel || index.Product == "" {
		return Index{}, ErrInvalidIndex
	}
	for target, release := range index.Targets {
		if _, err := ParseTarget(string(target)); err != nil || release.Version == "" {
			return Index{}, ErrInvalidIndex
		}
		for _, file := range []File{release.Installer, release.Update} {
			if _, err := ParseFileName(file.Name); err != nil || file.Size <= 0 || !sha256Pattern.MatchString(file.SHA256) {
				return Index{}, ErrInvalidIndex
			}
		}
		if channel == Stable && release.Prerelease {
			return Index{}, ErrInvalidIndex
		}
	}
	return index, nil
}
