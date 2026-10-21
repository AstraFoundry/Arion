package bootstrap

import (
	"bytes"
	"html/template"
	"strings"

	"arion.local/desktop/ui/shared/copy"
)

// appURLScheme is declared as urlSchemes in desktop.json. Opening a URL of
// it brings Arion to the front; the browser asks before it leaves the page.
const appURLScheme = "arion"

const appURL = appURLScheme + "://open"

// callbackPageTemplate is what the browser shows after the sign-in callback.
// The outcome is shown in Arion, so the page only leads back to it. The
// loopback answers with a CSP that allows no scripts.
var callbackPageTemplate = template.Must(template.New("callback").Parse(`<!doctype html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>{{.Title}}</title>
<style>
:root { --canvas: #f7f5f0; --text: #1c1a17; --muted: #6b665d; --accent: #ffcc00; --accent-hover: #ffd633; --accent-ink: #1c1a17; }
@media (prefers-color-scheme: dark) { :root { --canvas: #1c1a17; --text: #f4f1ea; --muted: #a8a196; } }
* { box-sizing: border-box; }
body { margin: 0; min-height: 100vh; display: grid; place-items: center; padding: 24px 16px; background: var(--canvas); color: var(--text);
  font: 15px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif; -webkit-font-smoothing: antialiased; }
main { max-width: 360px; text-align: center; }
h1 { margin: 0 0 8px; font-size: 22px; font-weight: 600; letter-spacing: -0.01em; }
p { margin: 0 0 24px; color: var(--muted); }
a { display: inline-block; padding: 10px 22px; border-radius: 8px; background: var(--accent); color: var(--accent-ink);
  font-weight: 600; text-decoration: none; transition: background-color 120ms ease-out; }
a:hover { background: var(--accent-hover); }
a:focus-visible { outline: 2px solid var(--text); outline-offset: 2px; }
</style>
</head>
<body>
<main>
<h1>{{.Title}}</h1>
<p>{{.Body}}</p>
<a href="{{.URL}}">{{.Open}}</a>
</main>
</body>
</html>
`))

// callbackPage renders the page in the interface language.
func callbackPage(locale copy.Locale) []byte {
	if locale == "" {
		locale = copy.English
	}
	messages := copy.For(locale)
	var buf bytes.Buffer
	err := callbackPageTemplate.Execute(&buf, struct {
		Lang, Title, Body, Open string
		URL                     template.URL
	}{
		Lang:  string(locale),
		Title: messages.Text(copy.AccCallbackTitle),
		Body:  messages.Text(copy.AccCallbackBody),
		Open:  messages.Text(copy.AccCallbackOpen),
		URL:   template.URL(appURL),
	})
	if err != nil {
		panic(err) // the template and its data are fixed
	}
	return buf.Bytes()
}

// isAppURL reports whether s is a URL of appURLScheme.
func isAppURL(s string) bool {
	scheme, _, ok := strings.Cut(s, ":")
	return ok && strings.EqualFold(scheme, appURLScheme)
}

// hasAppURL reports whether args, a second instance's arguments, carry a URL
// of appURLScheme; OnOpenURL handles those.
func hasAppURL(args []string) bool {
	for _, arg := range args {
		if isAppURL(arg) {
			return true
		}
	}
	return false
}
