//go:build windows

package imebridge

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"arion.local/core/domain"
)

var pipeClientPID = pipeKernel.NewProc("GetNamedPipeClientProcessId")
var processSession = pipeKernel.NewProc("ProcessIdToSessionId")

func peerFailure() error {
	return domain.Fail(domain.CodeArtifactUnverified, domain.DetailInputMethodArtifactUnverified)
}

func tokenBytes(token syscall.Token, class uint32) ([]byte, error) {
	var size uint32
	err := syscall.GetTokenInformation(token, class, nil, 0, &size)
	if (!errors.Is(err, syscall.ERROR_INSUFFICIENT_BUFFER) && !errors.Is(err, syscall.Errno(24))) || size == 0 {
		return nil, peerFailure()
	}
	value := make([]byte, size)
	if err = syscall.GetTokenInformation(token, class, &value[0], size, &size); err != nil {
		return nil, err
	}
	return value, nil
}

func ordinaryIdentity(token syscall.Token) (string, uint32, error) {
	user, err := token.GetTokenUser()
	if err != nil {
		return "", 0, err
	}
	sid, err := user.User.Sid.String()
	if err != nil {
		return "", 0, err
	}
	elevation, err := tokenBytes(token, syscall.TokenElevation)
	if err != nil || len(elevation) < 4 || *(*uint32)(unsafe.Pointer(&elevation[0])) != 0 {
		return "", 0, peerFailure()
	}
	label, err := tokenBytes(token, syscall.TokenIntegrityLevel)
	if err != nil || len(label) < int(unsafe.Sizeof(uintptr(0))) {
		return "", 0, peerFailure()
	}
	identity := *(**syscall.SID)(unsafe.Pointer(&label[0]))
	text, err := identity.String()
	if err != nil {
		return "", 0, err
	}
	last := strings.LastIndexByte(text, '-')
	if last < 0 {
		return "", 0, peerFailure()
	}
	integrity, err := strconv.ParseUint(text[last+1:], 10, 32)
	if err != nil || integrity < 8192 || integrity >= 12288 {
		return "", 0, peerFailure()
	}
	return sid, uint32(integrity), nil
}

func processSessionID(pid uint32) (uint32, error) {
	var session uint32
	if ok, _, err := processSession.Call(uintptr(pid), uintptr(unsafe.Pointer(&session))); ok == 0 {
		return 0, err
	}
	return session, nil
}

func (p pipeConnection) peerIdentity() (PeerIdentity, func(), error) {
	var pid uint32
	if ok, _, _ := pipeClientPID.Call(p.Fd(), uintptr(unsafe.Pointer(&pid))); ok == 0 || pid == 0 {
		return PeerIdentity{}, func() {}, peerFailure()
	}
	process, err := syscall.OpenProcess(0x1000|syscall.SYNCHRONIZE, false, pid)
	if err != nil {
		return PeerIdentity{}, func() {}, peerFailure()
	}
	var guard sync.Mutex
	closed := false
	release := func() {
		guard.Lock()
		defer guard.Unlock()
		if !closed {
			closed = true
			syscall.CloseHandle(process)
		}
	}
	fail := func() (PeerIdentity, func(), error) { release(); return PeerIdentity{}, func() {}, peerFailure() }
	var token syscall.Token
	if syscall.OpenProcessToken(process, syscall.TOKEN_QUERY, &token) != nil {
		return fail()
	}
	sid, integrity, err := ordinaryIdentity(token)
	token.Close()
	if err != nil {
		return fail()
	}
	ownProcess, err := syscall.GetCurrentProcess()
	if err != nil {
		return fail()
	}
	var own syscall.Token
	if syscall.OpenProcessToken(ownProcess, syscall.TOKEN_QUERY, &own) != nil {
		return fail()
	}
	ownSID, ownIntegrity, err := ordinaryIdentity(own)
	own.Close()
	if err != nil {
		return fail()
	}
	session, err := processSessionID(pid)
	if err != nil {
		return fail()
	}
	ownSession, err := processSessionID(uint32(syscall.Getpid()))
	if err != nil || ownSession != session || sid != ownSID || integrity != ownIntegrity {
		return fail()
	}
	peer := PeerIdentity{ProcessID: pid, SessionID: session, SID: sid, Integrity: integrity, alive: func() bool {
		guard.Lock()
		defer guard.Unlock()
		if closed {
			return false
		}
		state, err := syscall.WaitForSingleObject(process, 0)
		return err == nil && state == syscall.WAIT_TIMEOUT
	}}
	if !peer.Alive() {
		return fail()
	}
	return peer, release, nil
}
