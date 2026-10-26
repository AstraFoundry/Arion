// Package productnotifications implements the product inbox HTTP and local ledger ports.
package productnotifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	policy "arion.local/desktop/internal/productnotifications"
)

const maxResponse = 1 << 20

type Client struct {
	origin string
	http   *http.Client
	Now    func() time.Time
}

func NewClient(origin string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.Path != "" && u.Path != "/" {
		return nil, policy.ErrPayload
	}
	c := http.Client{Timeout: 20 * time.Second}
	if httpClient != nil {
		c = *httpClient
	}
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{origin: strings.TrimRight(origin, "/"), http: &c, Now: time.Now}, nil
}
func (c *Client) request(ctx context.Context, s policy.Session, method, path string, body any, out any) error {
	if !s.Valid() || s.Issuer != c.origin {
		return policy.ErrAuthorization
	}
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return policy.ErrPayload
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.origin+path, bytes.NewReader(data))
	if err != nil {
		return policy.ErrPayload
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return policy.ErrHTTP
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return policy.ErrAuthorization
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w_%d", policy.ErrHTTP, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(raw) > maxResponse {
		return policy.ErrPayload
	}
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) != nil || !envelope.Success || len(envelope.Data) == 0 || string(envelope.Data) == "null" || len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		return policy.ErrPayload
	}
	if json.Unmarshal(envelope.Data, out) != nil {
		return policy.ErrPayload
	}
	return nil
}
func (c *Client) Inbox(ctx context.Context, s policy.Session, cursor string) (policy.Page, error) {
	var p policy.Page
	if cursor != "" && !policy.ValidCursor(cursor) {
		return p, policy.ErrPayload
	}
	q := url.Values{"productSlug": {policy.Product}, "limit": {"20"}}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	err := c.request(ctx, s, http.MethodGet, "/api/v1/notifications?"+q.Encode(), nil, &p)
	if err != nil {
		return p, err
	}
	if err = policy.ValidatePage(p, c.Now()); err != nil {
		return policy.Page{}, err
	}
	return p, nil
}
func (c *Client) Receipt(ctx context.Context, s policy.Session, id, event, actionID string) (policy.Receipt, error) {
	var r policy.Receipt
	if !policy.ValidMessageID(id) {
		return r, policy.ErrPayload
	}
	if event != "received" && event != "read" && event != "dismissed" && event != "action" {
		return r, policy.ErrPayload
	}
	body := struct {
		ProductSlug string `json:"productSlug"`
		Event       string `json:"event"`
		ActionID    string `json:"actionId,omitempty"`
	}{policy.Product, event, actionID}
	if event == "action" && !policy.ValidActionID(actionID) || event != "action" && actionID != "" {
		return r, policy.ErrPayload
	}
	err := c.request(ctx, s, http.MethodPost, "/api/v1/notifications/"+url.PathEscape(id)+"/receipt", body, &r)
	return r, err
}
