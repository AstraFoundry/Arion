//go:build windows

package imebridge

import (
	"errors"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var pipeKernel = syscall.NewLazyDLL("kernel32.dll")
var pipeCreate = pipeKernel.NewProc("CreateNamedPipeW")
var pipeConnect = pipeKernel.NewProc("ConnectNamedPipe")
var pipeEvent = pipeKernel.NewProc("CreateEventW")
var pipeWait = pipeKernel.NewProc("WaitForSingleObject")
var pipeResult = pipeKernel.NewProc("GetOverlappedResult")
var pipeCancel = pipeKernel.NewProc("CancelIoEx")
var pipeAdvapi = syscall.NewLazyDLL("advapi32.dll")
var pipeSDDL = pipeAdvapi.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
var pipeFree = pipeKernel.NewProc("LocalFree")

// WindowsPipePath isolates the desktop listener by the current user's SID.
// The native TSF host derives the same name, without trusting environment paths.
func WindowsPipePath() (string, error) {
	var token syscall.Token
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		return "", err
	}
	if err := syscall.OpenProcessToken(process, syscall.TOKEN_QUERY, &token); err != nil {
		return "", err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	sid, err := user.User.Sid.String()
	if err != nil {
		return "", err
	}
	return `\\.\pipe\Arion.Voice.TSF.` + sid, nil
}

type pipeAddress string

func (a pipeAddress) Network() string { return "arion-pipe" }
func (a pipeAddress) String() string  { return string(a) }

type pipeConnection struct {
	*os.File
	address pipeAddress
}

func (p pipeConnection) LocalAddr() net.Addr           { return p.address }
func (p pipeConnection) RemoteAddr() net.Addr          { return p.address }
func (p pipeConnection) SetDeadline(t time.Time) error { return p.File.SetDeadline(t) }

type pipeListener struct {
	path    string
	mu      sync.Mutex
	closed  bool
	pending syscall.Handle
	first   bool
}

func listenSocket(path string) (net.Listener, func(), error) {
	l := &pipeListener{path: path, first: true}
	// Reserve the first instance before reporting success. A pre-existing pipe
	// must fail rather than sharing a name with an unrelated listener.
	h, err := l.create()
	if err != nil {
		return nil, nil, err
	}
	l.pending = h
	return l, func() {}, nil
}
func (l *pipeListener) create() (syscall.Handle, error) {
	var token syscall.Token
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, err
	}
	if err := syscall.OpenProcessToken(process, syscall.TOKEN_QUERY, &token); err != nil {
		return 0, err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return 0, err
	}
	sid, err := user.User.Sid.String()
	if err != nil {
		return 0, err
	}
	sddl, _ := syscall.UTF16PtrFromString("D:P(A;;GA;;;" + sid + ")")
	var descriptor uintptr
	if ok, _, e := pipeSDDL.Call(uintptr(unsafe.Pointer(sddl)), 1, uintptr(unsafe.Pointer(&descriptor)), 0); ok == 0 {
		return 0, e
	}
	defer pipeFree.Call(descriptor)
	sa := syscall.SecurityAttributes{Length: uint32(unsafe.Sizeof(syscall.SecurityAttributes{})), SecurityDescriptor: descriptor}
	name, err := syscall.UTF16PtrFromString(l.path)
	if err != nil {
		return 0, err
	}
	flags := uintptr(0x3 | 0x40000000) // duplex, overlapped
	if l.first {
		flags |= 0x00080000
	} // FILE_FLAG_FIRST_PIPE_INSTANCE
	h, _, e := pipeCreate.Call(uintptr(unsafe.Pointer(name)), flags, 0x8, 255, 65536, 65536, 0, uintptr(unsafe.Pointer(&sa))) // reject remote clients
	if h == ^uintptr(0) {
		return 0, e
	}
	l.first = false
	return syscall.Handle(h), nil
}
func (l *pipeListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, net.ErrClosed
	}
	h := l.pending
	if h == 0 {
		var err error
		h, err = l.create()
		if err != nil {
			l.mu.Unlock()
			return nil, err
		}
		l.pending = h
	}
	l.mu.Unlock()
	event, _, e := pipeEvent.Call(0, 1, 0, 0)
	if event == 0 {
		return nil, e
	}
	defer syscall.CloseHandle(syscall.Handle(event))
	o := syscall.Overlapped{HEvent: syscall.Handle(event)}
	ok, _, e := pipeConnect.Call(uintptr(h), uintptr(unsafe.Pointer(&o)))
	if ok == 0 && e == syscall.Errno(997) { // ERROR_IO_PENDING
		pipeWait.Call(event, 0xffffffff)
		var done uint32
		ok, _, e = pipeResult.Call(uintptr(h), uintptr(unsafe.Pointer(&o)), uintptr(unsafe.Pointer(&done)), 0)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, net.ErrClosed
	}
	l.pending = 0
	if ok == 0 && e != syscall.Errno(535) {
		syscall.CloseHandle(h)
		return nil, e
	} // ERROR_PIPE_CONNECTED
	f := os.NewFile(uintptr(h), l.path)
	if f == nil {
		syscall.CloseHandle(h)
		return nil, errors.New("named pipe handle is unavailable")
	}
	return pipeConnection{File: f, address: pipeAddress(l.path)}, nil
}
func (l *pipeListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	if l.pending != 0 {
		pipeCancel.Call(uintptr(l.pending), 0)
		_ = syscall.CloseHandle(l.pending)
		l.pending = 0
	}
	return nil
}
func (l *pipeListener) Addr() net.Addr { return pipeAddress(l.path) }
