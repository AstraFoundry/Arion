//go:build windows

package imebridge

import (
	"fmt"
	"os"
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
		t.Fatalf("fixture requires ordinary medium token: integrity=%d err=%v", integrity, identityErr)
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
