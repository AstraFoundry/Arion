package bootstrap

import (
	"strings"
	"testing"

	"arion.local/desktop/ui/shared/copy"
)

func TestCallbackPageLinksToApp(t *testing.T) {
	for _, locale := range []copy.Locale{"", copy.English, copy.SimplifiedChinese} {
		page := string(callbackPage(locale))
		messages := copy.For(locale)
		for _, want := range []string{`href="arion://open"`, messages.Text(copy.AccCallbackTitle), messages.Text(copy.AccCallbackBody), messages.Text(copy.AccCallbackOpen)} {
			if !strings.Contains(page, want) {
				t.Errorf("%q page lacks %q", locale, want)
			}
		}
		if strings.Contains(page, "<script") {
			t.Errorf("%q page has a script", locale)
		}
	}
	if !strings.Contains(string(callbackPage(copy.SimplifiedChinese)), `lang="zh-CN"`) {
		t.Error("Chinese page lacks its lang")
	}
}

func TestAppURL(t *testing.T) {
	for url, want := range map[string]bool{"arion://open": true, "ARION://open": true, "https://arion": false, "arion": false, `C:\arion`: false} {
		if got := isAppURL(url); got != want {
			t.Errorf("isAppURL(%q) = %v", url, got)
		}
	}
	if hasAppURL([]string{"--flag"}) || !hasAppURL([]string{"--flag", "arion://open"}) {
		t.Error("hasAppURL")
	}
}
