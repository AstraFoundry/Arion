package bootstrap

// desktopProfile separates mutable development state and OS credentials from releases.
func desktopProfile(development bool) string {
	if development {
		return "Arion Dev"
	}
	return "Arion"
}
