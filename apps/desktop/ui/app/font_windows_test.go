//go:build windows

package app

import (
	"fmt"
	"os"
	"testing"

	"arion.local/desktop/ui/shared/typography"
)

// Render the real bundled face in every Windows view test and capture.
func TestMain(m *testing.M) {
	if err := typography.Register(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
