//go:build windows

package imebridge

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestWindowsPipeProofUsesKernelClientAndHeldProcess(t *testing.T) {
	process, _ := syscall.GetCurrentProcess()
	var token syscall.Token
	if err := syscall.OpenProcessToken(process, syscall.TOKEN_QUERY, &token); err != nil {
		t.Fatal(err)
	}
	sid, integrity, identityErr := ordinaryIdentity(token)
	token.Close()
	if identityErr != nil || sid == "" || integrity != 8192 {
		// An elevated development shell is not a valid positive fixture.
		// Exercise the same kernel pipe proof in a real reduced-token child;
		// never relax ordinaryIdentity or turn this acceptance into a skip.
		if os.Getenv("ARION_PEER_PROOF_CHILD") == "1" {
			t.Fatalf("child requires ordinary medium token: integrity=%d err=%v", integrity, identityErr)
		}
		runMediumPeerProof(t)
		return
	}
	path := fmt.Sprintf(`\\.\pipe\Arion.PeerProof.Test.%d.%d`, os.Getpid(), time.Now().UnixNano())
	listener, release, err := listenSocket(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	defer listener.Close()
	result := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			result <- err
			return
		}
		defer connection.Close()
		peer, closePeer, err := inspectPeer(connection)
		if err != nil {
			result <- err
			return
		}
		if peer.ProcessID != uint32(os.Getpid()) || peer.SID == "" || peer.Integrity < 8192 || !peer.Alive() {
			closePeer()
			result <- fmt.Errorf("incorrect kernel identity: %+v", peer)
			return
		}
		closePeer()
		if peer.Alive() {
			result <- fmt.Errorf("released process handle remained valid")
			return
		}
		result <- nil
	}()
	name, _ := syscall.UTF16PtrFromString(path)
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(handle)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestLowIntegrityDuplicateTokenIsRejected(t *testing.T) {
	process, _ := syscall.GetCurrentProcess()
	var original syscall.Token
	if err := syscall.OpenProcessToken(process, syscall.TOKEN_QUERY|syscall.TOKEN_DUPLICATE, &original); err != nil {
		t.Fatal(err)
	}
	defer original.Close()
	var duplicate syscall.Token
	if ok, _, err := pipeAdvapi.NewProc("DuplicateTokenEx").Call(uintptr(original), syscall.TOKEN_QUERY|syscall.TOKEN_ADJUST_DEFAULT, 0, 2, 2, uintptr(unsafe.Pointer(&duplicate))); ok == 0 {
		t.Fatal(err)
	}
	defer duplicate.Close()
	sid, err := syscall.StringToSid("S-1-16-4096")
	if err != nil {
		t.Fatal(err)
	}
	label := syscall.SIDAndAttributes{Sid: sid, Attributes: 0x20}
	if ok, _, err := pipeAdvapi.NewProc("SetTokenInformation").Call(uintptr(duplicate), syscall.TokenIntegrityLevel, uintptr(unsafe.Pointer(&label)), unsafe.Sizeof(label)+uintptr(syscall.GetLengthSid(sid))); ok == 0 {
		t.Fatal(err)
	}
	// Only a duplicate token is modified, never the process/thread token.
	if _, _, err := ordinaryIdentity(duplicate); err == nil {
		t.Fatal("low integrity token accepted")
	}
}

// runMediumPeerProof lowers only a child token. The user's process, account,
// UAC configuration and desktop permissions are never modified.
func runMediumPeerProof(t *testing.T) {
	t.Helper()
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		t.Fatal(err)
	}
	var original syscall.Token
	access := uint32(syscall.TOKEN_QUERY | syscall.TOKEN_DUPLICATE | syscall.TOKEN_ADJUST_DEFAULT | syscall.TOKEN_ASSIGN_PRIMARY)
	if err := syscall.OpenProcessToken(process, access, &original); err != nil {
		t.Fatal(err)
	}
	defer original.Close()
	elevation, err := tokenBytes(original, syscall.TokenElevation)
	if err != nil || len(elevation) < 4 || *(*uint32)(unsafe.Pointer(&elevation[0])) != 1 {
		t.Fatalf("unexpected ordinary-token failure: elevation=%v err=%v", elevation, err)
	}
	if _, _, err := ordinaryIdentity(original); err == nil {
		t.Fatal("elevated parent token accepted as ordinary")
	}
	var reduced syscall.Token
	// DISABLE_MAX_PRIVILEGE | LUA_TOKEN strips administrative groups/privileges.
	if ok, _, err := pipeAdvapi.NewProc("CreateRestrictedToken").Call(uintptr(original), 0x1|0x4, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&reduced))); ok == 0 {
		t.Fatal(err)
	}
	defer reduced.Close()
	medium, err := syscall.StringToSid("S-1-16-8192")
	if err != nil {
		t.Fatal(err)
	}
	label := syscall.SIDAndAttributes{Sid: medium, Attributes: 0x20}
	if ok, _, err := pipeAdvapi.NewProc("SetTokenInformation").Call(uintptr(reduced), syscall.TokenIntegrityLevel, uintptr(unsafe.Pointer(&label)), unsafe.Sizeof(label)+uintptr(syscall.GetLengthSid(medium))); ok == 0 {
		t.Fatal(err)
	}
	if sid, integrity, err := ordinaryIdentity(reduced); err != nil || sid == "" || integrity != 8192 {
		t.Fatalf("reduced token failed identity: integrity=%d err=%v", integrity, err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestWindowsPipeProofUsesKernelClientAndHeldProcess$", "-test.v", "-test.timeout=30s")
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "ARION_PEER_PROOF_CHILD=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "ARION_PEER_PROOF_CHILD=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Token: reduced}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("medium-token pipe proof: %v\n%s", err, output)
	}
	t.Logf("ordinary medium-token subprocess:\n%s", output)
}
