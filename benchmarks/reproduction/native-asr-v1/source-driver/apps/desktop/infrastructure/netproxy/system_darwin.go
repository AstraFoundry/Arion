package netproxy

import (
	"bufio"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

var getenv = os.Getenv

// system reads the manual proxies of the active network service from
// `scutil --proxy`, the same dictionary CFNetwork uses. PAC files and
// automatic discovery are not evaluated.
func system() Settings {
	out, err := exec.Command("/usr/sbin/scutil", "--proxy").Output()
	if err != nil {
		return Settings{}
	}
	return parseScutil(string(out))
}

func parseScutil(text string) Settings {
	values := map[string]string{}
	var s Settings
	inExceptions := false
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if inExceptions {
			if line == "}" {
				inExceptions = false
				continue
			}
			if _, value, ok := strings.Cut(line, " : "); ok {
				s.Exceptions = append(s.Exceptions, strings.TrimSpace(value))
			}
			continue
		}
		key, value, ok := strings.Cut(line, " : ")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) == "ExceptionsList" {
			inExceptions = true
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	proxy := func(prefix string) *url.URL {
		if values[prefix+"Enable"] != "1" || values[prefix+"Proxy"] == "" {
			return nil
		}
		host := values[prefix+"Proxy"]
		if port := values[prefix+"Port"]; port != "" {
			host = net.JoinHostPort(host, port)
		}
		return &url.URL{Scheme: "http", Host: host}
	}
	s.HTTP, s.HTTPS = proxy("HTTP"), proxy("HTTPS")
	return s
}
