//go:build windows

// Package typography owns the bundled speech-interface font.
package typography

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/egoist/mygo/ui"
)

// Family is shared by the speech preview and transcript views.
const Family = "Source Han Sans SC"

//go:embed fonts/SourceHanSansSC-Regular.otf
var regular []byte

//go:embed fonts/SourceHanSansSC-Medium.otf
var medium []byte

var registration struct {
	sync.Once
	err error
}

// Register loads private app fonts without changing the Windows font registry.
func Register() error {
	registration.Do(func() {
		for _, data := range [][]byte{regular, medium} {
			if err := ui.RegisterFont(data, Family); err != nil {
				registration.err = fmt.Errorf("register speech font: %w", err)
				return
			}
		}
	})
	return registration.err
}
