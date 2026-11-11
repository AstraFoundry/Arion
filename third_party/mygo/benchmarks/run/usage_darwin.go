package main

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	procPidRusage func(pid, flavor int32, buf unsafe.Pointer) int32
	// cpuScale turns the CPU times of rusage_info into nanoseconds. Their unit
	// differs by architecture: found once, against getrusage.
	cpuScale float64
)

var (
	cgWindowListCopyWindowInfo func(option, relativeTo uint32) uintptr
	cfArrayGetCount            func(array uintptr) int
	cfArrayGetValueAtIndex     func(array uintptr, i int) uintptr
	cfDictionaryGetValue       func(dict, key uintptr) uintptr
	cfNumberGetValue           func(number uintptr, kind int, value *int32) bool
	cfRelease                  func(obj uintptr)
	kCGWindowOwnerPID          uintptr
)

func init() {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&procPidRusage, lib, "proc_pid_rusage")
	cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&cfArrayGetCount, cf, "CFArrayGetCount")
	purego.RegisterLibFunc(&cfArrayGetValueAtIndex, cf, "CFArrayGetValueAtIndex")
	purego.RegisterLibFunc(&cfDictionaryGetValue, cf, "CFDictionaryGetValue")
	purego.RegisterLibFunc(&cfNumberGetValue, cf, "CFNumberGetValue")
	purego.RegisterLibFunc(&cfRelease, cf, "CFRelease")
	cg, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&cgWindowListCopyWindowInfo, cg, "CGWindowListCopyWindowInfo")
	key, err := purego.Dlsym(cg, "kCGWindowOwnerPID")
	if err != nil {
		panic(err)
	}
	kCGWindowOwnerPID = **(**uintptr)(unsafe.Pointer(&key))
}

// windowOnScreen reports whether a window of pid is on screen, as the window
// server tells.
func windowOnScreen(pid int) bool {
	const onScreenOnly, cfNumberSInt32Type = 1, 3
	list := cgWindowListCopyWindowInfo(onScreenOnly, 0)
	if list == 0 {
		return false
	}
	defer cfRelease(list)
	for i := range cfArrayGetCount(list) {
		var owner int32
		number := cfDictionaryGetValue(cfArrayGetValueAtIndex(list, i), kCGWindowOwnerPID)
		if number != 0 && cfNumberGetValue(number, cfNumberSInt32Type, &owner) && int(owner) == pid {
			return true
		}
	}
	return false
}

// usage is what a process has used up to now.
type usage struct {
	cpu             time.Duration
	footprint, peak uint64
}

// usageOf reads rusage_info_v4 of pid, which keeps fields at these offsets
// after a 16-byte UUID (<sys/resource.h>).
func usageOf(pid int) (usage, error) {
	if cpuScale == 0 {
		calibrate()
	}
	user, system, footprint, peak, err := rusage(int32(pid))
	if err != nil {
		return usage{}, err
	}
	return usage{
		cpu:       time.Duration(float64(user+system) * cpuScale),
		footprint: footprint,
		peak:      peak,
	}, nil
}

func rusage(pid int32) (user, system, footprint, peak uint64, err error) {
	var info [512]byte
	if procPidRusage(pid, 4, unsafe.Pointer(&info[0])) != 0 {
		return 0, 0, 0, 0, fmt.Errorf("proc_pid_rusage(%d) failed", pid)
	}
	return binary.LittleEndian.Uint64(info[16:]), binary.LittleEndian.Uint64(info[24:]),
		binary.LittleEndian.Uint64(info[72:]), binary.LittleEndian.Uint64(info[240:]), nil
}

// calibrate spins for a moment, and compares this process's rusage_info with
// getrusage, which counts in microseconds everywhere.
func calibrate() {
	own := func() (ticks uint64, cpu time.Duration) {
		user, system, _, _, err := rusage(int32(syscall.Getpid()))
		if err != nil {
			panic(err)
		}
		var ru syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
			panic(err)
		}
		return user + system, time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
	}
	t0, c0 := own()
	for end := time.Now().Add(300 * time.Millisecond); time.Now().Before(end); {
	}
	t1, c1 := own()
	cpuScale = float64(c1-c0) / float64(t1-t0)
}
