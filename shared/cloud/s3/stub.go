package s3

import (
	"context"
	"io"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

// StubClient is a no-op ObjectStore implementation for use in test mode. FileExistsResult controls what FileExists reports: it defaults to false (so photo/label lookups behave as "absent"), but a service that needs uploads to validate as present in test mode (e.g. chat attachments) can set it to true.
type StubClient struct {
	FileExistsResult bool
	// GetResult is the byte payload returned by Get (e.g. a canned raw email for inbound-bridge tests).
	GetResult []byte
}

func (s *StubClient) Upload(_ context.Context, _, _ string, _ io.Reader, _ string) *apierror.APIError {
	return nil
}

func (s *StubClient) GetPresignedURL(_ context.Context, _, _ string, _ time.Duration) (string, *apierror.APIError) {
	return "https://stub.local/presigned", nil
}

func (s *StubClient) GetPresignedPutURL(_ context.Context, _, _, _ string, _ time.Duration) (string, *apierror.APIError) {
	return "https://stub.local/presigned-put", nil
}

func (s *StubClient) FileExists(_ context.Context, _, _ string) (bool, *apierror.APIError) {
	return s.FileExistsResult, nil
}

func (s *StubClient) Get(_ context.Context, _, _ string) ([]byte, *apierror.APIError) {
	return s.GetResult, nil
}

func (s *StubClient) Delete(_ context.Context, _, _ string) *apierror.APIError {
	return nil
}

func (s *StubClient) Copy(_ context.Context, _, _, _ string) *apierror.APIError {
	return nil
}
