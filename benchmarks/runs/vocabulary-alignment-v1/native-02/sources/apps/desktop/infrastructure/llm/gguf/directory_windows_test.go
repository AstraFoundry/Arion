package gguf

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

// PRV-A01: Windows socket files must inherit only the owner's directory ACL;
// POSIX chmod success alone does not establish this Windows boundary.
func TestPrivateDirectoryUsesProtectedCurrentUserACL(t *testing.T) {
	dir, err := privateDirectory()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	name, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		t.Fatal(err)
	}
	advapi := syscall.NewLazyDLL("advapi32.dll")
	get := advapi.NewProc("GetFileSecurityW")
	var needed uint32
	get.Call(uintptr(unsafe.Pointer(name)), 4, 0, 0, uintptr(unsafe.Pointer(&needed)))
	if needed == 0 {
		t.Fatal("security descriptor size unavailable")
	}
	buffer := make([]byte, needed)
	if ok, _, err := get.Call(uintptr(unsafe.Pointer(name)), 4, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), uintptr(unsafe.Pointer(&needed))); ok == 0 {
		t.Fatal(err)
	}
	var text *uint16
	convert := advapi.NewProc("ConvertSecurityDescriptorToStringSecurityDescriptorW")
	if ok, _, err := convert.Call(uintptr(unsafe.Pointer(&buffer[0])), 1, 4, uintptr(unsafe.Pointer(&text)), 0); ok == 0 {
		t.Fatal(err)
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(text)))
	var chars []uint16
	for p := unsafe.Pointer(text); ; p = unsafe.Add(p, 2) {
		v := *(*uint16)(p)
		if v == 0 {
			break
		}
		chars = append(chars, v)
	}
	acl := syscall.UTF16ToString(chars)
	var token syscall.Token
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.OpenProcessToken(process, syscall.TOKEN_QUERY, &token); err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid, err := user.User.Sid.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(acl, "D:P") || strings.Count(acl, "(") != 1 || !strings.Contains(acl, "FA;;;"+sid+")") || !strings.Contains(acl, "OICI") {
		t.Fatalf("unexpected directory ACL: %s", acl)
	}
}
