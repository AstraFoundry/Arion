// Package productnotifications owns account-scoped product notification policy.
// Notification payloads contain display data and validated HTTPS actions only.
package productnotifications

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const Product = "arion"
const NativePrefix = "arion.product."
const MaxRecords = 4096

var (
	ErrPayload       = errors.New("product_notification_payload")
	ErrDeferred      = errors.New("product_notification_deferred")
	ErrStale         = errors.New("product_notification_stale")
	ErrStore         = errors.New("product_notification_store")
	ErrAuthorization = errors.New("product_notification_authorization")
	ErrHTTP          = errors.New("product_notification_http")
	ErrOpen          = errors.New("product_notification_open")
	uuid             = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	actionID         = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
)

type Session struct{ Issuer, Subject, Token string }

func (s Session) Valid() bool {
	return s.Issuer != "" && s.Subject != "" && len(s.Subject) <= 512 && s.Token != "" && !strings.ContainsAny(s.Token, " \t\r\n")
}
func (s Session) Key() string {
	h := sha256.Sum256([]byte(s.Issuer + "\x00" + s.Subject + "\x00" + Product))
	return hex.EncodeToString(h[:])
}

type Button struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Action string `json:"action"`
	URL    string `json:"url,omitempty"`
}
type Receipt struct {
	ReceivedAt  *time.Time `json:"receivedAt"`
	ReadAt      *time.Time `json:"readAt"`
	DismissedAt *time.Time `json:"dismissedAt"`
	ActionIDs   []string   `json:"actionIds"`
}

func (r Receipt) Handled() bool {
	return r.ReceivedAt != nil || r.ReadAt != nil || r.DismissedAt != nil || len(r.ActionIDs) > 0
}

type Message struct {
	ID          string     `json:"id"`
	ProductSlug string     `json:"productSlug"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Buttons     []Button   `json:"buttons"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	PublishedAt time.Time  `json:"publishedAt"`
	Receipt     Receipt    `json:"receipt"`
}
type Page struct {
	Items              []Message `json:"items"`
	NextCursor         *string   `json:"nextCursor"`
	AllowedActionHosts []string  `json:"allowedActionHosts"`
	Enabled            bool      `json:"enabled"`
}

func text(s string, max int, multiline bool) bool {
	if s == "" || !utf8.ValidString(s) || utf8.RuneCountInString(s) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\r' || r == '\t')) {
			return false
		}
	}
	return true
}
func ValidHost(host string) bool {
	if host == "" || len(host) > 253 || host != strings.ToLower(host) || net.ParseIP(host) != nil || strings.HasSuffix(host, ".") {
		return false
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return false
	}
	for _, suffix := range []string{"localhost", "local", "internal", "lan", "home"} {
		if parts[len(parts)-1] == suffix {
			return false
		}
	}
	tld := parts[len(parts)-1]
	letters := false
	for _, r := range tld {
		if r >= 'a' && r <= 'z' {
			letters = true
		}
	}
	if !letters {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 || part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}
func ValidateURL(raw string, hosts []string) error {
	if raw == "" || len(raw) > 2048 || strings.ContainsAny(raw, "\r\n\t\\") {
		return ErrPayload
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.Host == "" || u.Port() != "" && u.Port() != "443" || !ValidHost(strings.ToLower(u.Hostname())) {
		return ErrPayload
	}
	for _, h := range hosts {
		if h == strings.ToLower(u.Hostname()) {
			return nil
		}
	}
	return ErrPayload
}
func ValidateMessage(m Message, hosts []string, now time.Time) error {
	if !uuid.MatchString(m.ID) || m.ProductSlug != Product || !text(m.Title, 120, false) || !text(m.Body, 2000, true) || m.PublishedAt.IsZero() || m.PublishedAt.After(now.Add(time.Minute)) || len(m.Buttons) > 2 {
		return ErrPayload
	}
	if m.ExpiresAt != nil && (m.ExpiresAt.IsZero() || !m.ExpiresAt.After(m.PublishedAt)) {
		return ErrPayload
	}
	seen := map[string]bool{}
	for _, b := range m.Buttons {
		if !actionID.MatchString(b.ID) || seen[b.ID] || !text(b.Label, 40, false) {
			return ErrPayload
		}
		seen[b.ID] = true
		switch b.Action {
		case "dismiss":
			if b.URL != "" {
				return ErrPayload
			}
		case "open_url":
			if ValidateURL(b.URL, hosts) != nil {
				return ErrPayload
			}
		default:
			return ErrPayload
		}
	}
	if len(m.Receipt.ActionIDs) > 2 {
		return ErrPayload
	}
	for _, id := range m.Receipt.ActionIDs {
		if !seen[id] {
			return ErrPayload
		}
	}
	return nil
}
func ValidatePage(p Page, now time.Time) error {
	if len(p.Items) > 20 || len(p.AllowedActionHosts) > 128 {
		return ErrPayload
	}
	if p.NextCursor != nil && !ValidCursor(*p.NextCursor) {
		return ErrPayload
	}
	for _, h := range p.AllowedActionHosts {
		if !ValidHost(h) {
			return ErrPayload
		}
	}
	seen := map[string]bool{}
	for _, m := range p.Items {
		if seen[m.ID] || ValidateMessage(m, p.AllowedActionHosts, now) != nil {
			return ErrPayload
		}
		seen[m.ID] = true
	}
	return nil
}
func NativeID(accountKey, messageID string) string {
	h := sha256.Sum256([]byte(accountKey + "\x00" + messageID))
	return NativePrefix + hex.EncodeToString(h[:])
}
func ValidMessageID(id string) bool { return uuid.MatchString(id) }
func ValidActionID(id string) bool  { return actionID.MatchString(id) }

func ValidCursor(cursor string) bool {
	if len(cursor) == 0 || len(cursor) > 256 {
		return false
	}
	for _, r := range cursor {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// UnmarshalJSON requires the published inbox contract. Missing configuration
// fields are API failures, never interpreted as a disabled notification service.
func (p *Page) UnmarshalJSON(data []byte) error {
	var raw struct {
		Items              *[]Message      `json:"items"`
		NextCursor         json.RawMessage `json:"nextCursor"`
		AllowedActionHosts *[]string       `json:"allowedActionHosts"`
		Enabled            *bool           `json:"enabled"`
	}
	if json.Unmarshal(data, &raw) != nil || raw.Items == nil || raw.AllowedActionHosts == nil || raw.Enabled == nil || len(raw.NextCursor) == 0 {
		return ErrPayload
	}
	var cursor *string
	if json.Unmarshal(raw.NextCursor, &cursor) != nil {
		return ErrPayload
	}
	*p = Page{Items: *raw.Items, NextCursor: cursor, AllowedActionHosts: *raw.AllowedActionHosts, Enabled: *raw.Enabled}
	return nil
}
