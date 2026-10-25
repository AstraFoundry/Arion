package subprocess

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Child creation is deliberately gated on stdin, matching the pipe worker's
// invariant that it cannot spawn the runtime before Job Object assignment.
func TestJobOwnsDescendants(t *testing.T) {
	if mode := os.Getenv("ARION_JOB_TEST"); mode != "" {
		if mode == "parent" {
			_, _ = bufio.NewReader(os.Stdin).ReadByte()
			child := exec.Command(os.Args[0], "-test.run=^TestJobOwnsDescendants$")
			child.Env = append(os.Environ(), "ARION_JOB_TEST=child")
			if err := child.Start(); err != nil {
				os.Exit(2)
			}
			fmt.Fprintln(os.Stdout, child.Process.Pid)
		}
		for {
			time.Sleep(time.Second)
		}
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestJobOwnsDescendants$")
	cmd.Env = append(os.Environ(), "ARION_JOB_TEST=parent")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	process, err := Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer process.Stop()
	_, _ = stdin.Write([]byte{1})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(handle)
	process.Stop()
	if result, err := syscall.WaitForSingleObject(handle, 5000); err != nil || result != syscall.WAIT_OBJECT_0 {
		t.Fatalf("descendant survived owner close: %d %v", result, err)
	}
	_ = cmd.Wait()
}
