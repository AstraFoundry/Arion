package clipboardos

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

type sourceFileAccessError struct{ cause error }

func (e *sourceFileAccessError) Error() string { return e.cause.Error() }
func (e *sourceFileAccessError) Unwrap() error { return e.cause }

// FileAccessDenied recognizes permission evidence from source-file operations,
// never an arbitrary history-store or provider path error.
func FileAccessDenied(err error) bool {
	var source *sourceFileAccessError
	return errors.As(err, &source)
}

func sourceFileError(err error) error {
	if FileAccessDenied(err) {
		return err
	}
	var path *os.PathError
	if runtime.GOOS == "darwin" && errors.Is(err, os.ErrPermission) && errors.As(err, &path) && filepath.IsAbs(path.Path) {
		return &sourceFileAccessError{cause: err}
	}
	return err
}
