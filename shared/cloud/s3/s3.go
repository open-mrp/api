package s3

import (
	"context"
	"errors"
	"io"
	"time"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectStore defines the interface for object storage operations.
type ObjectStore interface {
	Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string) *apierror.APIError
	GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, *apierror.APIError)
	FileExists(ctx context.Context, bucket, key string) (bool, *apierror.APIError)
}

var s3Tracer = tracing.GetTracer("shared.cloud.s3")

const defaultPresignExpiry = 1 * time.Hour

// Client wraps the AWS S3 SDK client with convenience methods.
type Client struct {
	client    *s3.Client
	presigner *s3.PresignClient
}

// NewClient creates a new S3 client using the given AWS region.
func NewClient(ctx context.Context, region string) (*Client, *apierror.APIError) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to load AWS configuration for S3.")
	}

	client := s3.NewFromConfig(cfg)
	return &Client{
		client:    client,
		presigner: s3.NewPresignClient(client),
	}, nil
}

// Upload uploads a file to S3.
func (c *Client) Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string) *apierror.APIError {
	ctx, span := s3Tracer.Start(ctx, "s3.upload")
	defer span.End()

	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String(key),
		Body:                 body,
		ContentType:          aws.String(contentType),
		ServerSideEncryption: s3types.ServerSideEncryptionAes256,
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to upload file to S3."))
	}

	return nil
}

// GetPresignedURL generates a presigned GET URL for an S3 object.
func (c *Client) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, *apierror.APIError) {
	ctx, span := s3Tracer.Start(ctx, "s3.get_presigned_url")
	defer span.End()

	if expiry == 0 {
		expiry = defaultPresignExpiry
	}

	result, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate presigned URL."))
	}

	return result.URL, nil
}

// FileExists checks whether an object exists in S3.
func (c *Client) FileExists(ctx context.Context, bucket, key string) (bool, *apierror.APIError) {
	ctx, span := s3Tracer.Start(ctx, "s3.file_exists")
	defer span.End()

	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *s3types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		// S3 HeadObject returns a 404 as a smithy HTTP response error.
		// It returns 403 instead of 404 when the caller lacks s3:ListBucket
		// permission, so treat both as "not found".
		var respErr interface{ HTTPStatusCode() int }
		if errors.As(err, &respErr) {
			if code := respErr.HTTPStatusCode(); code == 404 || code == 403 {
				return false, nil
			}
		}
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check S3 file existence."))
	}

	return true, nil
}
