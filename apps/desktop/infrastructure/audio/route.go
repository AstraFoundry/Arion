package audio

import (
	"strings"
	"unicode/utf16"

	"github.com/gen2brain/malgo"
)

// Endpoint is one capture device as the route chooser sees it.
// Headset means opening it puts Bluetooth headphones into the call
// profile. BuiltIn is a microphone in the computer, not a USB or
// Bluetooth accessory.
type Endpoint struct {
	ID      string
	Name    string
	Default bool
	Headset bool
	BuiltIn bool
}

// ResolveCaptureID chooses which endpoint to open (DSK-A23).
//
// selected empty means the system default. When that default is a
// headset microphone and a built-in microphone is present, the built-in
// microphone is used. Otherwise the first non-headset microphone is
// used. The returned empty string means "open the system default".
// An explicit selection is returned unchanged, including when it names
// a headset or a device that is not in the list.
//
// Opening the headset microphone switches it from the music profile to
// the call profile. macOS then shows the volume HUD for that output
// device, and Windows shows the volume flyout. Capture does not change
// the OS default input or either OS volume.
func ResolveCaptureID(selected string, devices []Endpoint) string {
	if selected != "" {
		return selected
	}
	var def *Endpoint
	for i := range devices {
		if devices[i].Default {
			def = &devices[i]
			break
		}
	}
	if def == nil || !def.Headset {
		return ""
	}
	for i := range devices {
		if devices[i].BuiltIn && !devices[i].Headset && devices[i].ID != "" {
			return devices[i].ID
		}
	}
	for i := range devices {
		if !devices[i].Headset && devices[i].ID != "" {
			return devices[i].ID
		}
	}
	return ""
}

type captureLabel struct {
	headset bool
	builtIn bool
}

func endpointsFrom(infos []malgo.DeviceInfo) []Endpoint {
	labels := captureLabels(infos)
	out := make([]Endpoint, len(infos))
	for i := range infos {
		out[i] = Endpoint{
			ID:      infos[i].ID.String(),
			Name:    infos[i].Name(),
			Default: infos[i].IsDefault != 0,
			Headset: labels[i].headset,
			BuiltIn: labels[i].builtIn,
		}
	}
	return out
}

// coreAudioUID reads the CoreAudio device UID stored at the front of a
// miniaudio device id.
func coreAudioUID(raw []byte) string {
	n := 0
	for n < len(raw) && raw[n] != 0 {
		n++
	}
	return string(raw[:n])
}

// wasapiEndpointID reads the UTF-16 WASAPI endpoint id at the front of a
// miniaudio device id. The field holds 64 wchar values.
func wasapiEndpointID(raw []byte) string {
	limit := 64 * 2
	if len(raw) < limit {
		limit = len(raw) - len(raw)%2
	}
	units := make([]uint16, 0, 32)
	for i := 0; i+1 < limit; i += 2 {
		c := uint16(raw[i]) | uint16(raw[i+1])<<8
		if c == 0 {
			break
		}
		units = append(units, c)
	}
	return string(utf16.Decode(units))
}

// endpointClass reports a Windows capture endpoint from its enumerator
// name and instance id. An unrecognized endpoint is neither a headset
// nor built-in, so the route chooser leaves the system default alone.
func endpointClass(enumerator, instanceID string) (headset, builtIn bool) {
	en := strings.ToUpper(strings.TrimSpace(enumerator))
	inst := strings.ToUpper(instanceID)
	if en == "BTHENUM" || en == "BTHHFENUM" ||
		strings.Contains(inst, "BTHENUM") || strings.Contains(inst, "BTHHFENUM") || strings.Contains(inst, "BTHLE") {
		return true, false
	}
	switch en {
	case "HDAUDIO", "INTELAUDIO", "INTCAUDIO":
		return false, true
	default:
		return false, false
	}
}
