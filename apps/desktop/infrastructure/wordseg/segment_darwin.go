//go:build darwin

// Package wordseg supplies local OS word boundaries for correction learning.
package wordseg

import (
	"fmt"
	"runtime"
	"sync"
	"unicode/utf16"

	"arion.local/core/domain"
	"github.com/ebitengine/purego"
)

// Segmenter uses Core Foundation's linguistic word tokenizer. Each call owns
// its tokenizer; only the immutable function bindings are shared.
type Segmenter struct{}

type cfRange struct{ Location, Length int64 }

var api struct {
	once            sync.Once
	err             error
	stringCreate    func(uintptr, []uint16, int64) uintptr
	tokenizerCreate func(uintptr, uintptr, cfRange, uint64, uintptr) uintptr
	advance         func(uintptr) uint64
	tokenRange      func(uintptr) cfRange
	release         func(uintptr)
}

func load() {
	handle, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		api.err = fmt.Errorf("load word tokenizer: %w", err)
		return
	}
	for _, binding := range []struct {
		name   string
		target any
	}{
		{"CFStringCreateWithCharacters", &api.stringCreate},
		{"CFStringTokenizerCreate", &api.tokenizerCreate},
		{"CFStringTokenizerAdvanceToNextToken", &api.advance},
		{"CFStringTokenizerGetCurrentTokenRange", &api.tokenRange},
		{"CFRelease", &api.release},
	} {
		symbol, err := purego.Dlsym(handle, binding.name)
		if err != nil {
			api.err = fmt.Errorf("bind word tokenizer: %w", err)
			return
		}
		purego.RegisterFunc(binding.target, symbol)
	}
	// The framework stays loaded for the lifetime of these function pointers.
}

// Words converts the OS's UTF-16 offsets to domain code-point offsets.
func (Segmenter) Words(text string) ([]domain.WordSpan, error) {
	if text == "" {
		return nil, nil
	}
	api.once.Do(load)
	if api.err != nil {
		return nil, api.err
	}
	encoded := utf16.Encode([]rune(text))
	str := api.stringCreate(0, encoded, int64(len(encoded)))
	runtime.KeepAlive(encoded)
	if str == 0 {
		return nil, fmt.Errorf("create tokenizer text")
	}
	defer api.release(str)
	tokenizer := api.tokenizerCreate(0, str, cfRange{0, int64(len(encoded))}, 0, 0)
	if tokenizer == 0 {
		return nil, fmt.Errorf("create word tokenizer")
	}
	defer api.release(tokenizer)
	offsets := make([]int, len(encoded)+1)
	point, unit := 0, 0
	for _, r := range text {
		offsets[unit] = point
		if r > 0xffff {
			offsets[unit+1] = -1
			unit += 2
		} else {
			unit++
		}
		point++
	}
	offsets[unit] = point
	var spans []domain.WordSpan
	for api.advance(tokenizer) != 0 {
		bounds := api.tokenRange(tokenizer)
		end := bounds.Location + bounds.Length
		if bounds.Location < 0 || bounds.Length <= 0 || end > int64(len(encoded)) || offsets[bounds.Location] < 0 || offsets[end] < 0 {
			return nil, fmt.Errorf("invalid tokenizer range")
		}
		spans = append(spans, domain.WordSpan{Start: offsets[bounds.Location], End: offsets[end]})
	}
	return spans, nil
}
