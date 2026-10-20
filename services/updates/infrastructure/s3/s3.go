// Package s3 reads the release bucket of Amazon S3 or a compatible service
// (Cloudflare R2, MinIO) with AWS Signature Version 4 requests.
package s3

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"arion.local/updates/application"
)

// emptyHash is the SHA-256 of an empty payload.
const emptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// Config names a bucket and read-only credentials.
type Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	Prefix          string
	PathStyle       bool
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// Bucket implements application.Bucket.
type Bucket struct {
	config   Config
	endpoint *url.URL
	prefix   string
	client   *http.Client
	now      func() time.Time
}

var _ application.Bucket = (*Bucket)(nil)

// New returns a Bucket. client and now default to http.DefaultClient and
// time.Now.
func New(config Config, client *http.Client, now func() time.Time) (*Bucket, error) {
	endpoint, err := url.Parse(config.Endpoint)
	if err != nil || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.Host == "" || strings.Trim(endpoint.Path, "/") != "" {
		return nil, fmt.Errorf("s3 endpoint must be an http(s) URL without a path")
	}
	if config.Bucket == "" || config.Region == "" || config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return nil, fmt.Errorf("s3 bucket, region and credentials are required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	if now == nil {
		now = time.Now
	}
	prefix := strings.Trim(config.Prefix, "/")
	if prefix != "" {
		prefix += "/"
	}
	return &Bucket{config: config, endpoint: endpoint, prefix: prefix, client: client, now: now}, nil
}

// objectURL returns the host and escaped path of key.
func (b *Bucket) objectURL(key string) (host, path string) {
	host, path = b.endpoint.Host, "/"+escape(b.prefix+key)
	if b.config.PathStyle {
		path = "/" + escape(b.config.Bucket) + path
	} else {
		host = b.config.Bucket + "." + host
	}
	return host, path
}

// Get implements application.Bucket.
func (b *Bucket) Get(ctx context.Context, key string, limit int64) ([]byte, error) {
	host, path := b.objectURL(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.endpoint.Scheme+"://"+host+path, nil)
	if err != nil {
		return nil, err
	}
	b.signHeaders(req, host, path, b.now())
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 get: %w", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, application.ErrNotFound
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("s3 get returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("s3 get: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("s3 object exceeds %d bytes", limit)
	}
	return data, nil
}

// PresignGet implements application.Bucket with a query-signed URL.
func (b *Bucket) PresignGet(key string, ttl time.Duration, now time.Time) (string, error) {
	seconds := int64(ttl / time.Second)
	if seconds < 1 || seconds > 7*24*3600 {
		return "", fmt.Errorf("presigned URL lifetime must be between 1 second and 7 days")
	}
	host, path := b.objectURL(key)
	now = now.UTC()
	date, amzDate := now.Format("20060102"), now.Format("20060102T150405Z")
	scope := date + "/" + b.config.Region + "/s3/aws4_request"
	query := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Credential":    b.config.AccessKeyID + "/" + scope,
		"X-Amz-Date":          amzDate,
		"X-Amz-Expires":       strconv.FormatInt(seconds, 10),
		"X-Amz-SignedHeaders": "host",
	}
	if b.config.SessionToken != "" {
		query["X-Amz-Security-Token"] = b.config.SessionToken
	}
	canonicalQuery := canonicalQueryString(query)
	request := "GET\n" + path + "\n" + canonicalQuery + "\nhost:" + host + "\n\nhost\nUNSIGNED-PAYLOAD"
	signature := b.signature(date, amzDate, scope, request)
	return b.endpoint.Scheme + "://" + host + path + "?" + canonicalQuery + "&X-Amz-Signature=" + signature, nil
}

// signHeaders signs a GET request without a body or query.
func (b *Bucket) signHeaders(req *http.Request, host, path string, now time.Time) {
	now = now.UTC()
	date, amzDate := now.Format("20060102"), now.Format("20060102T150405Z")
	headers := map[string]string{
		"host":                 host,
		"x-amz-content-sha256": emptyHash,
		"x-amz-date":           amzDate,
	}
	if b.config.SessionToken != "" {
		headers["x-amz-security-token"] = b.config.SessionToken
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	var canonical strings.Builder
	for _, name := range names {
		canonical.WriteString(name + ":" + headers[name] + "\n")
		if name != "host" {
			req.Header.Set(name, headers[name])
		}
	}
	signed := strings.Join(names, ";")
	scope := date + "/" + b.config.Region + "/s3/aws4_request"
	request := req.Method + "\n" + path + "\n\n" + canonical.String() + "\n" + signed + "\n" + emptyHash
	signature := b.signature(date, amzDate, scope, request)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+b.config.AccessKeyID+"/"+scope+
		", SignedHeaders="+signed+", Signature="+signature)
}

func (b *Bucket) signature(date, amzDate, scope, canonicalRequest string) string {
	hash := sha256.Sum256([]byte(canonicalRequest))
	key := hmacSHA256([]byte("AWS4"+b.config.SecretAccessKey), date)
	for _, part := range []string{b.config.Region, "s3", "aws4_request"} {
		key = hmacSHA256(key, part)
	}
	return hex.EncodeToString(hmacSHA256(key, "AWS4-HMAC-SHA256\n"+amzDate+"\n"+scope+"\n"+hex.EncodeToString(hash[:])))
}

func canonicalQueryString(query map[string]string) string {
	names := make([]string, 0, len(query))
	for name := range query {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = escapeComponent(name) + "=" + escapeComponent(query[name])
	}
	return strings.Join(parts, "&")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

// escape escapes a key for a URL path as Signature Version 4 requires:
// every byte but letters, digits, "-._~" and "/".
func escape(s string) string { return escapeExcept(s, "-._~/") }

// escapeComponent escapes a query name or value, "/" included.
func escapeComponent(s string) string { return escapeExcept(s, "-._~") }

func escapeExcept(s, unreserved string) string {
	var out strings.Builder
	for i := range len(s) {
		ch := s[i]
		if 'A' <= ch && ch <= 'Z' || 'a' <= ch && ch <= 'z' || '0' <= ch && ch <= '9' || strings.IndexByte(unreserved, ch) >= 0 {
			out.WriteByte(ch)
		} else {
			fmt.Fprintf(&out, "%%%02X", ch)
		}
	}
	return out.String()
}
