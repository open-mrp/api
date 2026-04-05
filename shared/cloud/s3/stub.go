package s3

import (
	"context"
	"io"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

// StubClient is a no-op ObjectStore implementation for use in test mode.
type StubClient struct{}

func (s *StubClient) Upload(_ context.Context, _, _ string, _ io.Reader, _ string) *apierror.APIError {
	return nil
}

func (s *StubClient) GetPresignedURL(_ context.Context, _, _ string, _ time.Duration) (string, *apierror.APIError) {
	return "https://stub.local/presigned", nil
}

func (s *StubClient) FileExists(_ context.Context, _, _ string) (bool, *apierror.APIError) {
	return false, nil
}
