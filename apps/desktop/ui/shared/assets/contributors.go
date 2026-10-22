package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"

	"github.com/egoist/mygo/ui"
)

//go:embed contributors
var contributorFiles embed.FS

// Contributor is a GitHub contributor to Arion, with the avatar GitHub
// showed when scripts/update-contributors.mjs last ran.
type Contributor struct {
	Login   string
	Profile string
	Avatar  *ui.Bitmap
}

// Contributors lists the repository's contributors in GitHub's order, most
// contributions first. The app embeds them; it never asks GitHub.
var Contributors = mustContributors()

func mustContributors() []Contributor {
	data, err := contributorFiles.ReadFile("contributors/contributors.json")
	if err != nil {
		panic("assets: " + err.Error())
	}
	var list struct {
		Contributors []struct {
			Login, Profile, Avatar string
			AvatarSHA256           string `json:"avatarSHA256"`
		}
	}
	if err := json.Unmarshal(data, &list); err != nil {
		panic("assets: contributors.json: " + err.Error())
	}
	out := make([]Contributor, 0, len(list.Contributors))
	for _, c := range list.Contributors {
		bytes, err := contributorFiles.ReadFile("contributors/" + c.Avatar)
		if err != nil {
			panic("assets: " + err.Error())
		}
		if sum := sha256.Sum256(bytes); hex.EncodeToString(sum[:]) != c.AvatarSHA256 {
			panic("assets: avatar of " + c.Login + " does not match contributors.json")
		}
		out = append(out, Contributor{Login: c.Login, Profile: c.Profile, Avatar: mustDecode(bytes)})
	}
	return out
}
