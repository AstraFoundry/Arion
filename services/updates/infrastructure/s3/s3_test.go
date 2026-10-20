package s3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"arion.local/updates/application"
)

// The presigned URL example of the Amazon S3 Signature Version 4
// documentation ("Authenticating Requests: Using Query Parameters").
func TestPresignMatchesAWSExample(t *testing.T) {
	bucket, err := New(Config{
		Endpoint:        "https://s3.amazonaws.com",
		Region:          "us-east-1",
		Bucket:          "examplebucket",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := bucket.PresignGet("test.txt", 86400*time.Second, time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	want := "https://examplebucket.s3.amazonaws.com/test.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAIOSFODNN7EXAMPLE%2F20130524%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20130524T000000Z&X-Amz-Expires=86400&X-Amz-SignedHeaders=host&X-Amz-Signature=aeeed9bbccd4d02ee5c0109b86d86835f995330da4c265957d157751f604d404"
	if got != want {
		t.Fatalf("presigned URL\n got %s\nwant %s", got, want)
	}
	if _, err := bucket.PresignGet("test.txt", 8*24*time.Hour, time.Now()); err == nil {
		t.Fatal("a lifetime beyond seven days was accepted")
	}
}

func TestGetPathStyleWithPrefix(t *testing.T) {
	var seen *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		switch r.URL.Path {
		case "/releases/arion/channels/beta/index.json":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/releases/arion/big":
			_, _ = w.Write([]byte(strings.Repeat("x", 20)))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	bucket, err := New(Config{Endpoint: server.URL, Region: "auto", Bucket: "releases", Prefix: "/arion/", PathStyle: true, AccessKeyID: "id", SecretAccessKey: "secret"}, server.Client(), func() time.Time { return time.Unix(0, 0) })
	if err != nil {
		t.Fatal(err)
	}
	data, err := bucket.Get(context.Background(), "channels/beta/index.json", 1024)
	if err != nil || string(data) != `{"ok":true}` {
		t.Fatalf("get = %q, %v", data, err)
	}
	if auth := seen.Header.Get("Authorization"); !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=id/19700101/auto/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=") {
		t.Fatalf("authorization = %q", auth)
	}
	if _, err := bucket.Get(context.Background(), "missing", 1024); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("missing object: %v", err)
	}
	if _, err := bucket.Get(context.Background(), "big", 10); err == nil {
		t.Fatal("an oversized object was read")
	}
}

func TestConfigRejected(t *testing.T) {
	for _, config := range []Config{
		{Endpoint: "https://s3.example/path", Region: "auto", Bucket: "b", AccessKeyID: "a", SecretAccessKey: "s"},
		{Endpoint: "ftp://s3.example", Region: "auto", Bucket: "b", AccessKeyID: "a", SecretAccessKey: "s"},
		{Endpoint: "https://s3.example", Region: "auto", Bucket: "b"},
	} {
		if _, err := New(config, nil, nil); err == nil {
			t.Errorf("accepted %+v", config)
		}
	}
}
