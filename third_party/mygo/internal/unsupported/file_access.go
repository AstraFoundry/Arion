package unsupported

import (
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/transfer"
)

func (clipboard) WithFileAccess(data transfer.Data, read func() error) error {
	return platform.WithLocalFileAccess(data, read)
}
