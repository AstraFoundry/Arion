//go:build darwin

package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

// ElementMain is the macOS 12 name. Older headers still have ElementMaster,
// and both values are 0.
#ifndef kAudioObjectPropertyElementMain
#define kAudioObjectPropertyElementMain 0
#endif

// 0 unlabeled, 1 Bluetooth headset, 2 built-in. A query failure stays 0
// so capture does not guess a route from the device name.
static int arion_capture_class(const char *uid) {
    AudioObjectPropertyAddress addr;
    UInt32 size = 0;
    OSStatus st;
    AudioObjectID *ids;
    UInt32 count, i;
    CFStringRef want;
    int class = 0;

    if (uid == NULL || uid[0] == '\0') return 0;
    addr.mSelector = kAudioHardwarePropertyDevices;
    addr.mScope = kAudioObjectPropertyScopeGlobal;
    addr.mElement = kAudioObjectPropertyElementMain;
    st = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr, 0, NULL, &size);
    if (st != noErr || size == 0 || size > 4096 * sizeof(AudioObjectID)) return 0;
    ids = (AudioObjectID *)malloc(size);
    if (ids == NULL) return 0;
    st = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, ids);
    if (st != noErr) {
        free(ids);
        return 0;
    }
    count = size / sizeof(AudioObjectID);
    want = CFStringCreateWithCString(NULL, uid, kCFStringEncodingUTF8);
    if (want == NULL) {
        free(ids);
        return 0;
    }
    for (i = 0; i < count; i++) {
        CFStringRef got = NULL;
        UInt32 n = sizeof(got);
        Boolean eq;
        UInt32 transport = 0;
        addr.mSelector = kAudioDevicePropertyDeviceUID;
        if (AudioObjectGetPropertyData(ids[i], &addr, 0, NULL, &n, &got) != noErr || got == NULL) continue;
        eq = CFStringCompare(got, want, 0) == kCFCompareEqualTo;
        CFRelease(got);
        if (!eq) continue;
        n = sizeof(transport);
        addr.mSelector = kAudioDevicePropertyTransportType;
        if (AudioObjectGetPropertyData(ids[i], &addr, 0, NULL, &n, &transport) != noErr) break;
        if (transport == kAudioDeviceTransportTypeBluetooth || transport == kAudioDeviceTransportTypeBluetoothLE) class = 1;
        else if (transport == kAudioDeviceTransportTypeBuiltIn) class = 2;
        break;
    }
    CFRelease(want);
    free(ids);
    return class;
}
*/
import "C"

import (
	"unsafe"

	"github.com/gen2brain/malgo"
)

func captureLabels(infos []malgo.DeviceInfo) []captureLabel {
	out := make([]captureLabel, len(infos))
	for i := range infos {
		uid := coreAudioUID(infos[i].ID[:])
		if uid == "" {
			continue
		}
		switch darwinCaptureClass(uid) {
		case 1:
			out[i].headset = true
		case 2:
			out[i].builtIn = true
		}
	}
	return out
}

func darwinCaptureClass(uid string) int {
	c := C.CString(uid)
	defer C.free(unsafe.Pointer(c))
	return int(C.arion_capture_class(c))
}
