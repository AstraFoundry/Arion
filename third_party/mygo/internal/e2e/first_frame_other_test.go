//go:build !js && !(windows && (amd64 || arm64))

package e2e

import "github.com/egoist/mygo"

// firstFrameRevealed reports whether w shows its first frame once shown:
// only Windows keeps a window out of sight after it showed (cloaked).
func firstFrameRevealed(*mygo.Window) bool { return true }
