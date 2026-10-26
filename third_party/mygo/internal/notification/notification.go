// Package notification owns bounded, platform-independent native wire formats.
package notification

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/internal/platform"
)

const activationPrefix = "mygo-notify:v1?"

// ActivatorCLSID is stable across upgrades and architectures for one sender.
func ActivatorCLSID(appID string) string {
	sum := sha256.Sum256([]byte("mygo.notification.activator.v1:" + appID))
	sum[6] = sum[6]&15 | 0x50
	sum[8] = sum[8]&63 | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

// Tag stays within Windows' 16-character tag limit and remains removable
// without retaining an in-memory notification object after a cold launch.
func Tag(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:8])
}

func ValidID(id string) bool {
	if id == "" || len(id) > 256 || !utf8.ValidString(id) {
		return false
	}
	for _, r := range id {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func ValidAction(a string) bool {
	if a == "" || len(a) > 64 {
		return false
	}
	for _, r := range a {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func Activation(id, action string) string {
	return activationPrefix + url.Values{"id": {id}, "action": {action}}.Encode()
}

// ParseActivation accepts only our versioned ID/action envelope. A notification
// never carries a command, URL, executable path or authorization to install.
func ParseActivation(value string) (id, action string, err error) {
	if len(value) > 2048 || !strings.HasPrefix(value, activationPrefix) {
		return "", "", errors.New("mygo: invalid notification activation")
	}
	q, err := url.ParseQuery(strings.TrimPrefix(value, activationPrefix))
	if err != nil || len(q) != 2 || len(q["id"]) != 1 || len(q["action"]) != 1 {
		return "", "", errors.New("mygo: invalid notification activation fields")
	}
	id, action = q.Get("id"), q.Get("action")
	if !ValidID(id) || action != "" && !ValidAction(action) {
		return "", "", errors.New("mygo: invalid notification activation values")
	}
	return id, action, nil
}

// ToastXML is a native COM-activated toast; no protocol activation or balloon
// transport is substituted if registration or WinRT fails.
func ToastXML(n *platform.Notification) (string, error) {
	if !ValidID(n.ID) || len(n.Actions) > 4 {
		return "", errors.New("mygo: invalid notification")
	}
	escape := func(s string) string { var b bytes.Buffer; _ = xml.EscapeText(&b, []byte(s)); return b.String() }
	var b strings.Builder
	fmt.Fprintf(&b, `<toast activationType="foreground" launch="%s"><visual><binding template="ToastGeneric"><text>%s</text>`, escape(Activation(n.ID, "")), escape(n.Title))
	if n.Subtitle != "" {
		fmt.Fprintf(&b, "<text>%s</text>", escape(n.Subtitle))
	}
	fmt.Fprintf(&b, "<text>%s</text></binding></visual>", escape(n.Body))
	if n.Silent {
		b.WriteString(`<audio silent="true"/>`)
	}
	if len(n.Actions) > 0 {
		b.WriteString("<actions>")
		for _, a := range n.Actions {
			if !ValidAction(a.ID) || a.Title == "" {
				return "", errors.New("mygo: invalid notification action")
			}
			// Desktop COM callbacks decide whether to raise a window. Foreground here
			// selects COM activation; it does not focus an application window itself.
			fmt.Fprintf(&b, `<action activationType="foreground" content="%s" arguments="%s"/>`, escape(a.Title), escape(Activation(n.ID, a.ID)))
		}
		b.WriteString("</actions>")
	}
	b.WriteString("</toast>")
	return b.String(), nil
}
