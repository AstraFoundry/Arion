//go:build darwin && cgo

package imebridge

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation -lbsm
#include "peer_darwin.h"
*/
import "C"

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"arion.local/core/domain"
)

// The code reference is located using the socket's kernel audit token, not a
// self-reported PID or a PID-to-path lookup. Release serializes with checks so
// retained callbacks cannot use a freed reference after connection teardown.
func inspectPlatformPeer(conn net.Conn) (PeerIdentity, func(), error) {
	failure := func() (PeerIdentity, func(), error) {
		return PeerIdentity{}, func() {}, domain.Fail(domain.CodeArtifactUnverified, domain.DetailInputMethodArtifactUnverified)
	}
	socket, ok := conn.(syscall.Conn)
	if !ok {
		return failure()
	}
	raw, err := socket.SyscallConn()
	if err != nil {
		return failure()
	}
	var code C.SecCodeRef
	var pid, uid C.uint32_t
	err = raw.Control(func(fd uintptr) { code = C.arion_socket_peer(C.int(fd), &pid, &uid) })
	if err != nil || code == 0 {
		if code != 0 {
			C.CFRelease(C.CFTypeRef(code))
		}
		return failure()
	}
	var mu sync.Mutex
	var publisher C.SecRequirementRef
	check := func(rule *C.char) bool {
		mu.Lock()
		defer mu.Unlock()
		if code == 0 || (publisher != 0 && C.arion_peer_matches(code, publisher) == 0) {
			return false
		}
		return C.arion_peer_valid(code, rule) != 0
	}
	release := func() {
		mu.Lock()
		defer mu.Unlock()
		if code != 0 {
			C.CFRelease(C.CFTypeRef(code))
			code = 0
		}
		if publisher != 0 {
			C.CFRelease(C.CFTypeRef(publisher))
			publisher = 0
		}
	}
	peer := PeerIdentity{ProcessID: uint32(pid), SID: strconv.FormatUint(uint64(uid), 10), alive: func() bool { return check(nil) }}
	peer.matchesCodeRequirement = func(rule string) bool {
		if rule == "" || len(rule) > 16384 || strings.ContainsRune(rule, 0) {
			return false
		}
		text := C.CString(rule)
		defer C.free(unsafe.Pointer(text))
		return check(text)
	}
	peer.bindPublisher = func(identifier string) bool {
		if identifier == "" || len(identifier) > 255 || strings.ContainsRune(identifier, 0) {
			return false
		}
		name := C.CString(identifier)
		defer C.free(unsafe.Pointer(name))
		requirement := C.arion_peer_publisher_requirement(name)
		if requirement == 0 {
			return false
		}
		mu.Lock()
		defer mu.Unlock()
		if code == 0 || C.arion_peer_matches(code, requirement) == 0 {
			C.CFRelease(C.CFTypeRef(requirement))
			return false
		}
		if publisher != 0 {
			C.CFRelease(C.CFTypeRef(publisher))
		}
		publisher = requirement
		return true
	}
	return peer, release, nil
}
