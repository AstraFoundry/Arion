//go:build windows && (amd64 || arm64)

package windows

import (
	"slices"
	"syscall"
	"testing"
	"unsafe"
)

// TestCommandLine checks that the arguments of a jump list task come back
// as they were from the command line Windows splits.
func TestCommandLine(t *testing.T) {
	args := []string{"myapp://new?a=1&b=2", "two words", `C:\Program Files\x\`, `say "hi"`, "", `back\\slash`}
	line := u16("app.exe " + commandLine(args))
	var argc int32
	argv, err := syscall.CommandLineToArgv(line, &argc)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.LocalFree(syscall.Handle(uintptr(unsafe.Pointer(argv))))
	var got []string
	for i := 1; i < int(argc); i++ {
		got = append(got, syscall.UTF16ToString(argv[i][:]))
	}
	if !slices.Equal(got, args) {
		t.Errorf("CommandLineToArgv(%q) = %q, want %q", commandLine(args), got, args)
	}
}
