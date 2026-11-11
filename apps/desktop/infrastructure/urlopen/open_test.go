package urlopen

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"arion.local/core/application"
)

func TestOpenRefusesSchemesWithoutLaunching(t *testing.T) {
	const secret = "SecretSelection"
	for _, raw := range []string{
		"javascript:alert(" + secret + ")",
		"file:///tmp/" + secret,
		"http://example.com/?q=" + secret,
		"chatgpt://new?prompt=" + secret,
		"doubao://chat?q=" + secret,
		"feishu://applink?q=" + secret,
	} {
		err := Open(raw)
		if !errors.Is(err, application.ErrSelectionScheme) || strings.Contains(err.Error(), secret) {
			t.Fatalf("%s: %v", raw, err)
		}
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return
	}
	err := Open("claude://claude.ai/new?q=" + secret)
	if err == nil || errors.Is(err, application.ErrSelectionScheme) || strings.Contains(err.Error(), secret) {
		t.Fatal(err)
	}
}
