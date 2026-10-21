// Package browser opens an http(s) URL in the system browser.
package browser

import (
	"errors"
	"net/url"
)

// Open validates rawURL and starts the platform opener. It does not wait for
// the browser to exit on Windows or Linux; on macOS it waits for /usr/bin/open,
// which returns after launch.
func Open(rawURL string) error {
	if err := Validate(rawURL); err != nil {
		return err
	}
	return open(rawURL)
}

// Validate accepts absolute http and https URLs only.
func Validate(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return errors.New("browser url must be http or https")
	}
	return nil
}
