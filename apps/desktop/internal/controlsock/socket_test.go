package controlsock

import (
	"net"
	"path/filepath"
	"testing"
)

func TestListenIsOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	prepareSharedParent(t, dir)
	path := filepath.Join(dir, "control.sock")
	listener, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	requireOwnerOnly(t, path)
	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
