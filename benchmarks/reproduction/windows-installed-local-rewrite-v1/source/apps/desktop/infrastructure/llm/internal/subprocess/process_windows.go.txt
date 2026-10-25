package subprocess

import (
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel    = syscall.NewLazyDLL("kernel32.dll")
	createJob = kernel.NewProc("CreateJobObjectW")
	setJob    = kernel.NewProc("SetInformationJobObject")
	assignJob = kernel.NewProc("AssignProcessToJobObject")
)

type basicLimits struct {
	ProcessTime, JobTime                 int64
	Flags                                uint32
	MinimumWorkingSet, MaximumWorkingSet uintptr
	ActiveProcesses                      uint32
	Affinity                             uintptr
	Priority, Scheduling                 uint32
}

type jobLimits struct {
	Basic                                                      basicLimits
	IO                                                         [6]uint64
	ProcessMemory, JobMemory, PeakProcessMemory, PeakJobMemory uintptr
}

// Process holds a noninheritable kill-on-close Job Object. The pipe worker must
// wait for its first request before spawning children: Start assigns membership
// before the caller can send that request. The native server spawns no children.
type Process struct {
	job  syscall.Handle
	once sync.Once
}

func Start(cmd *exec.Cmd) (*Process, error) {
	job, _, err := createJob.Call(0, 0)
	if job == 0 {
		return nil, err
	}
	p := &Process{job: syscall.Handle(job)}
	limits := jobLimits{Basic: basicLimits{Flags: 0x2000}} // JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if ok, _, err := setJob.Call(job, 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits)); ok == 0 {
		p.Stop()
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	if err := cmd.Start(); err != nil {
		p.Stop()
		return nil, err
	}
	handle, err := syscall.OpenProcess(0x100|0x1, false, uint32(cmd.Process.Pid)) // SET_QUOTA | TERMINATE
	if err == nil {
		ok, _, assignErr := assignJob.Call(job, uintptr(handle))
		syscall.CloseHandle(handle)
		if ok == 0 {
			err = assignErr
		}
	}
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		p.Stop()
		return nil, err
	}
	return p, nil
}

func (p *Process) Stop() { p.once.Do(func() { _ = syscall.CloseHandle(p.job) }) }

func StartChild(cmd *exec.Cmd) (*Process, error) { return Start(cmd) }
