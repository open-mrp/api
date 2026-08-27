package s3_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/cloud/s3"
)

// A presigned GET is loaded by a browser <img> tag, which sends nothing but Host. Any other signed header — the SDK adds `x-amz-checksum-mode` by default — makes S3 reject the signature with a 403.
func TestGetPresignedURLSignsOnlyHost(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("S3_ENDPOINT_URL", "")

	client, apiErr := s3.NewClient(context.Background(), "us-east-2")
	if apiErr != nil {
		t.Fatalf("NewClient: %v", apiErr)
	}

	raw, apiErr := client.GetPresignedURL(context.Background(), "bucket", "branding/favicon.png", time.Hour)
	if apiErr != nil {
		t.Fatalf("GetPresignedURL: %v", apiErr)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if got := parsed.Query().Get("X-Amz-SignedHeaders"); got != "host" {
		t.Errorf("X-Amz-SignedHeaders = %q, want %q", got, "host")
	}
}
