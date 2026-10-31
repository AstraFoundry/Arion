//go:build windows

package controlsock

import (
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// prepareSharedParent grants Everyone an inheritable allow ACE. Listen must
// drop that grant. A private temp directory would hide a chmod-only fix,
// because Windows already limits that directory to the current user.
func prepareSharedParent(t *testing.T, dir string) {
	t.Helper()
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid := user.User.Sid.String()
	if sid == "" {
		t.Fatal("current user SID is empty")
	}
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;OICI;GA;;;WD)(A;OICI;GA;;;" + sid + ")")
	if err != nil {
		t.Fatal(err)
	}
	dacl, defaulted, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if dacl == nil || defaulted {
		t.Fatal("shared parent security descriptor has no DACL")
	}
	if err := windows.SetNamedSecurityInfo(
		dir,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	); err != nil {
		t.Fatal(err)
	}
	runtime.KeepAlive(descriptor)
}

func requireOwnerOnly(t *testing.T, path string) {
	t.Helper()
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := descriptor.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatalf("socket DACL is inheriting, control %#x", control)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if dacl == nil || dacl.AceCount != 1 {
		count := 0
		if dacl != nil {
			count = int(dacl.AceCount)
		}
		t.Fatalf("socket DACL ace count %d, want 1", count)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
		t.Fatalf("socket ACE type %d, want allow", ace.Header.AceType)
	}
	if ace.Header.AceFlags != 0 {
		t.Fatalf("socket ACE flags %#x, want an explicit ACE", ace.Header.AceFlags)
	}
	// GENERIC_ALL is what Listen writes. Windows may map it to FILE_ALL_ACCESS
	// or to the file generic-write mask when it stores the ACE.
	const fileAllAccess windows.ACCESS_MASK = 0x001F01FF
	writable := ace.Mask == windows.GENERIC_ALL ||
		ace.Mask == fileAllAccess ||
		ace.Mask&windows.FILE_GENERIC_WRITE == windows.FILE_GENERIC_WRITE ||
		ace.Mask&windows.GENERIC_WRITE == windows.GENERIC_WRITE
	if !writable {
		t.Fatalf("socket ACE mask %#x does not grant write", ace.Mask)
	}
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if !user.User.Sid.Equals(aceSID) {
		t.Fatalf("socket ACE sid %s, want %s", aceSID.String(), user.User.Sid.String())
	}
	runtime.KeepAlive(descriptor)
}
