//go:build !darwin && !windows

package audio

import "github.com/gen2brain/malgo"

// captureLabels leaves every device unlabeled. Linux and other hosts
// have no headset-transport query; dictation keeps the selected device.
func captureLabels(infos []malgo.DeviceInfo) []captureLabel {
	return make([]captureLabel, len(infos))
}
