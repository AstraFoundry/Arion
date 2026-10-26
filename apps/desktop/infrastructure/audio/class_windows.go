//go:build windows

package audio

import (
	"syscall"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// Windows labels capture endpoints from the MMDevice enumerator.
// BTHENUM / BTHHFENUM is the hands-free microphone that switches the
// headset out of stereo and surfaces the volume flyout. HDAUDIO and
// Intel SST enumerators are the built-in microphone.

var (
	ole32                   = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx      = ole32.NewProc("CoInitializeEx")
	procCoUninitialize      = ole32.NewProc("CoUninitialize")
	procCoCreateInstance    = ole32.NewProc("CoCreateInstance")
	procPropVariantClear    = ole32.NewProc("PropVariantClear")
	clsidMMDeviceEnumerator = guid{0xbcde0395, 0xe52f, 0x467c, [8]byte{0x8e, 0x3d, 0xc4, 0x57, 0x92, 0x91, 0x69, 0x2e}}
	iidIMMDeviceEnumerator  = guid{0xa95664d2, 0x9614, 0x4f35, [8]byte{0xa7, 0x46, 0xde, 0x8d, 0xb6, 0x36, 0x17, 0xe6}}
	keyEnumeratorName       = propertyKey{guid{0xa45c254e, 0xdf1c, 0x4efd, [8]byte{0x80, 0x20, 0x67, 0xd1, 0x46, 0xa8, 0x50, 0xe0}}, 24}
	keyInstanceID           = propertyKey{guid{0x78c34fc8, 0x104a, 0x4aca, [8]byte{0x9e, 0xa4, 0x52, 0x4d, 0x52, 0x99, 0x6e, 0x57}}, 256}
)

const (
	clsctxAll           = 0x17
	coinitMultithreaded = 0
	vtLPWSTR            = 31
	vtBSTR              = 8
	rpcEChangedMode     = 0x80010106
)

type guid struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

type propertyKey struct {
	fmtid guid
	pid   uint32
}

// propVariant matches the 64-bit PROPVARIANT: the string pointer sits at
// offset 8. pad keeps PropVariantClear from reading past the value.
type propVariant struct {
	vt        uint16
	reserved1 uint16
	reserved2 uint16
	reserved3 uint16
	ptr       uintptr
	pad       uintptr
}

func captureLabels(infos []malgo.DeviceInfo) []captureLabel {
	out := make([]captureLabel, len(infos))
	for i := range infos {
		id := wasapiEndpointID(infos[i].ID[:])
		if id == "" {
			continue
		}
		enumerator, instance, ok := windowsEndpointProps(id)
		if !ok {
			continue
		}
		out[i].headset, out[i].builtIn = endpointClass(enumerator, instance)
	}
	return out
}

// windowsEndpointProps reads PKEY_Device_EnumeratorName and
// PKEY_Device_InstanceId. ok is false when the endpoint cannot be
// opened; the caller then leaves that device unlabeled.
func windowsEndpointProps(endpointID string) (enumerator, instance string, ok bool) {
	hr, _, _ := procCoInitializeEx.Call(0, coinitMultithreaded)
	code := uint32(hr)
	if int32(code) < 0 && code != rpcEChangedMode {
		return "", "", false
	}
	// Balance only the reference this call added. miniaudio keeps its own.
	if code == 0 || code == 1 {
		defer procCoUninitialize.Call()
	}
	var enumeratorObj uintptr
	if failed(coCreateEnumerator(&enumeratorObj)) {
		return "", "", false
	}
	defer release(enumeratorObj)

	id, err := syscall.UTF16PtrFromString(endpointID)
	if err != nil {
		return "", "", false
	}
	var device uintptr
	if failed(comCall(enumeratorObj, 5, uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(&device)))) {
		return "", "", false
	}
	defer release(device)

	var store uintptr
	if failed(comCall(device, 4, 0, uintptr(unsafe.Pointer(&store)))) {
		return "", "", false
	}
	defer release(store)
	return propString(store, keyEnumeratorName), propString(store, keyInstanceID), true
}

func propString(store uintptr, key propertyKey) string {
	var pv propVariant
	if failed(comCall(store, 5, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&pv)))) {
		return ""
	}
	defer clearPropVariant(&pv)
	if (pv.vt != vtLPWSTR && pv.vt != vtBSTR) || pv.ptr == 0 {
		return ""
	}
	p := (*uint16)(native(pv.ptr))
	n := 0
	for n < 1024 && *(*uint16)(unsafe.Add(unsafe.Pointer(p), n*2)) != 0 {
		n++
	}
	return syscall.UTF16ToString(unsafe.Slice(p, n))
}

func failed(hr uintptr) bool { return int32(uint32(hr)) < 0 }

//go:uintptrescapes
func coCreateEnumerator(out *uintptr) uintptr {
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidMMDeviceEnumerator)),
		0,
		clsctxAll,
		uintptr(unsafe.Pointer(&iidIMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(out)),
	)
	return hr
}

//go:uintptrescapes
func clearPropVariant(pv *propVariant) {
	procPropVariantClear.Call(uintptr(unsafe.Pointer(pv)))
}

func release(obj uintptr) {
	if obj != 0 {
		comCall(obj, 2)
	}
}

func native(addr uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&addr)) }

// comCall calls method i of the COM interface obj.
//
//go:uintptrescapes
func comCall(obj uintptr, i int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(native(obj))
	fn := *(*uintptr)(unsafe.Add(native(vtbl), i*int(unsafe.Sizeof(uintptr(0)))))
	var a [8]uintptr
	a[0] = obj
	n := copy(a[1:], args)
	r, _, _ := syscall.SyscallN(fn, a[:n+1]...)
	return r
}
