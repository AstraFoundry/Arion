package bootstrap

import "errors"

var (
	errInvalidReplacement     = errors.New("replacement needs both texts")
	errVocabularyUnavailable  = errors.New("vocabulary store unavailable")
	errInputMethodUnsupported = errors.New("no input method host on this platform")
)
