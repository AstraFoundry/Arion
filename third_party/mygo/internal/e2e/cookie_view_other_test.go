//go:build !js && !(windows && (amd64 || arm64))

package e2e

// cookieViewApp is TestCookieViewReleased's helper, on Windows only.
func cookieViewApp() {}

// webViewProcesses is WebView2's processes, which run on Windows only.
func webViewProcesses() []int { return nil }
