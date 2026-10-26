package audio

import (
	"testing"
	"unicode/utf16"
)

func TestResolveCaptureID(t *testing.T) {
	builtIn := Endpoint{ID: "builtin", Name: "MacBook Pro Microphone", BuiltIn: true}
	usb := Endpoint{ID: "usb", Name: "Desk Mic"}
	headset := Endpoint{ID: "bt", Name: "OPPO Enco Air3 Pro", Default: true, Headset: true}
	otherHeadset := Endpoint{ID: "bt2", Name: "Other buds", Headset: true}

	cases := []struct {
		name     string
		selected string
		devices  []Endpoint
		want     string
	}{
		{name: "default built-in stays system default", devices: []Endpoint{{ID: "builtin", Default: true, BuiltIn: true}}, want: ""},
		{name: "default usb stays system default", devices: []Endpoint{asDefault(usb)}, want: ""},
		{name: "headset default uses built-in", devices: []Endpoint{headset, usb, builtIn}, want: "builtin"},
		{name: "headset default later in the list uses built-in", devices: []Endpoint{builtIn, headset, usb}, want: "builtin"},
		{name: "headset default uses usb when no built-in", devices: []Endpoint{headset, usb}, want: "usb"},
		{name: "headset default with only headsets stays system default", devices: []Endpoint{headset, otherHeadset}, want: ""},
		{name: "no devices", devices: nil, want: ""},
		{name: "explicit headset is kept", selected: "bt", devices: []Endpoint{headset, builtIn}, want: "bt"},
		{name: "explicit built-in is kept", selected: "builtin", devices: []Endpoint{headset, builtIn}, want: "builtin"},
		{name: "missing explicit id is kept for the caller", selected: "gone", devices: []Endpoint{headset, builtIn}, want: "gone"},
		{name: "empty substitute id is skipped", devices: []Endpoint{headset, {Name: "nameless", BuiltIn: true}, usb}, want: "usb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveCaptureID(tc.selected, tc.devices); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func asDefault(e Endpoint) Endpoint {
	e.Default = true
	return e
}

func TestEndpointClass(t *testing.T) {
	cases := []struct {
		en, inst       string
		headset, local bool
	}{
		{"BTHENUM", "", true, false},
		{"bthhfenum", `BTHHFENUM\{0000111e-0000-1000-8000-00805f9b34fb}`, true, false},
		{"SWD", `SWD\MMDEVAPI\BTHLEDEVICE_1`, true, false},
		{"HDAUDIO", `HDAUDIO\FUNC_01&VEN_10EC`, false, true},
		{"INTELAUDIO", "", false, true},
		{"USB", `USB\VID_046D`, false, false},
		{"", "", false, false},
	}
	for _, tc := range cases {
		headset, local := endpointClass(tc.en, tc.inst)
		if headset != tc.headset || local != tc.local {
			t.Fatalf("%s / %s: got headset=%v built-in=%v", tc.en, tc.inst, headset, local)
		}
	}
}

func TestDeviceIDText(t *testing.T) {
	raw := make([]byte, 256)
	copy(raw, "BuiltInMicrophoneDevice")
	if got := coreAudioUID(raw); got != "BuiltInMicrophoneDevice" {
		t.Fatalf("uid %q", got)
	}
	id := `{0.0.1.00000000}.{aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee}`
	units := utf16.Encode([]rune(id))
	buf := make([]byte, 256)
	for i, c := range units {
		buf[i*2] = byte(c)
		buf[i*2+1] = byte(c >> 8)
	}
	if got := wasapiEndpointID(buf); got != id {
		t.Fatalf("endpoint %q", got)
	}
}
