// Package usageupload implements the daily usage HTTP client and local ledger.
package usageupload

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

	policy "arion.local/desktop/internal/usageupload"
)

const maxResponse = 1 << 20

type Client struct {
	origin string
	http   *http.Client
}

func NewClient(origin string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.Path != "" && u.Path != "/" {
		return nil, policy.ErrPayload
	}
	c := http.Client{Timeout: 20 * time.Second}
	if httpClient != nil {
		c = *httpClient
	}
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{origin: strings.TrimRight(origin, "/"), http: &c}, nil
}

func (c *Client) Report(ctx context.Context, s policy.Session, sourceID string, day policy.Day) (policy.Day, error) {
	var out policy.Day
	if len(sourceID) != 32 {
		return out, policy.ErrPayload
	}
	if !s.Valid() || s.Issuer != c.origin || !day.Valid() {
		return out, policy.ErrAuthorization
	}
	body := struct {
		SourceID    string `json:"sourceId"`
		ProductSlug string `json:"productSlug"`
		Date        string `json:"date"`
		Visited     bool   `json:"visited"`
		Sessions    int    `json:"sessions"`
		SpeechMs    int64  `json:"speechMs"`
		Words       int    `json:"words"`
	}{sourceID, policy.Product, day.Date, day.Visited, day.Sessions, day.SpeechMs, day.Words}
	data, err := json.Marshal(body)
	if err != nil {
		return out, policy.ErrPayload
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.origin+"/api/v1/usage", bytes.NewReader(data))
	if err != nil {
		return out, policy.ErrPayload
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		return out, policy.ErrHTTP
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return out, policy.ErrAuthorization
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("%w_%d", policy.ErrHTTP, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(raw) > maxResponse {
		return out, policy.ErrPayload
	}
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) != nil || !envelope.Success || len(envelope.Data) == 0 || string(envelope.Data) == "null" || len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		return out, policy.ErrPayload
	}
	if json.Unmarshal(envelope.Data, &out) != nil || !out.Valid() || out.Date != day.Date {
		return policy.Day{}, policy.ErrPayload
	}
	return out, nil
}
