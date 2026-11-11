package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestMain runs the test binary as an app that panics as it starts, for
// TestExitedAfterStarting.
func TestMain(m *testing.M) {
	if os.Getenv("IDLEMEM_TEST_PANIC") == "1" {
		panic("the app cannot start")
	}
	os.Exit(m.Run())
}

// TestExitedAfterStarting checks that an app that exits by itself before
// it is measured is told with what it wrote to its standard error, as the
// panic that ended it.
func TestExitedAfterStarting(t *testing.T) {
	t.Setenv("IDLEMEM_TEST_PANIC", "1")
	_, _, err := measure(os.Args[0], time.Hour)
	if err == nil || !strings.Contains(err.Error(), "exited after starting: exit status 2") || !strings.Contains(err.Error(), "panic: the app cannot start") {
		t.Fatalf("measure: %v, want the app's exit status and its panic", err)
	}
}
