//go:build !windows

package typography

// Family retains the platform interface face outside Windows.
const Family = ""

// Register has no bundled fonts to load on this platform.
func Register() error { return nil }
