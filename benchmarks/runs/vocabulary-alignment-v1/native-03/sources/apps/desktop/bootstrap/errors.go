package bootstrap

import "errors"

var (
	errVocabularyUnavailable  = errors.New("vocabulary store unavailable")
	errInputMethodUnsupported = errors.New("no input method host on this platform")
)
