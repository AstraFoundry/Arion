//go:build windows

package imebridge

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestWindowsPipeUsesRealIOAndRefusesCompetingListener(t *testing.T) {
	path := fmt.Sprintf(`\\.\pipe\Arion.Voice.TSF.Test.%d.%d`, os.Getpid(), time.Now().UnixNano())
	listener, release, err := listenSocket(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	defer listener.Close()
	if other, _, err := listenSocket(path); err == nil {
		other.Close()
		t.Fatal("shared a protected listener name")
	}
	result := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			result <- err
			return
		}
		defer conn.Close()
		if err = conn.SetReadDeadline(time.Now().Add(time.Second)); err == nil {
			var line string
			line, err = bufio.NewReader(conn).ReadString('\n')
			if err == nil && line != "hello\n" {
				err = fmt.Errorf("unexpected pipe bytes %q", line)
			}
		}
		if err == nil {
			_, err = conn.Write([]byte("ack\n"))
		}
		result <- err
	}()
	name, _ := syscall.UTF16PtrFromString(path)
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	client := os.NewFile(uintptr(handle), path)
	defer client.Close()
	if _, err := client.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	if line, err := bufio.NewReader(client).ReadString('\n'); err != nil || line != "ack\n" {
		t.Fatalf("pipe response %q: %v", line, err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}
