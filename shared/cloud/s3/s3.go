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
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectStore defines the interface for object storage operations.
type ObjectStore interface {
	Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string) *apierror.APIError
	GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, *apierror.APIError)
	// GetPresignedPutURL generates a presigned PUT URL so a client can upload directly to the bucket, keeping large media off the API request path.
	GetPresignedPutURL(ctx context.Context, bucket, key, contentType string, expiry time.Duration) (string, *apierror.APIError)
	FileExists(ctx context.Context, bucket, key string) (bool, *apierror.APIError)
	// Get fetches an object's full bytes. Used to read a raw inbound email the SES receipt rule stored.
	Get(ctx context.Context, bucket, key string) ([]byte, *apierror.APIError)
	// Delete removes an object. It is idempotent: deleting an already-absent key is not an error, so a reaper can re-attempt after a crash between the object delete and the DB row delete.
	Delete(ctx context.Context, bucket, key string) *apierror.APIError
	// Copy copies an object within a bucket (srcKey → dstKey). Used to promote a staged upload to its permanent key on attach; it is idempotent (re-copying overwrites the destination).
	Copy(ctx context.Context, bucket, srcKey, dstKey string) *apierror.APIError
}

var s3Tracer = tracing.GetTracer("shared.cloud.s3")

const defaultPresignExpiry = 1 * time.Hour

// Client wraps the AWS S3 SDK client with convenience methods.
type Client struct {
	client    *s3.Client
	presigner *s3.PresignClient
}

// NewClient creates a new S3 client using the given AWS region.
//
// The EC2 IMDS credential provider is disabled: pods receive credentials via IRSA (web identity token), never the node instance role, and IMDS is unreachable from pods anyway (the node hop limit is pinned to 1 as MCP hardening). Leaving IMDS in the credential chain makes credential resolution block for the full request deadline (~5s) on every call when no other provider is configured, rather than failing fast.
func NewClient(ctx context.Context, region string) (*Client, *apierror.APIError) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithEC2IMDSClientEnableState(imds.ClientDisabled),
	)
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

// GetPresignedPutURL generates a presigned PUT URL for direct client upload. The client must send the same Content-Type header when uploading. SSE-AES256 is applied on the bucket default.
func (c *Client) GetPresignedPutURL(ctx context.Context, bucket, key, contentType string, expiry time.Duration) (string, *apierror.APIError) {
	ctx, span := s3Tracer.Start(ctx, "s3.get_presigned_put_url")
	defer span.End()

	if expiry == 0 {
		expiry = defaultPresignExpiry
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	result, err := c.presigner.PresignPutObject(ctx, input, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate presigned upload URL."))
	}

	return result.URL, nil
}

// Delete removes an object from S3. S3 DeleteObject is idempotent — it returns success for a missing key — so this is safe to re-run after a partial purge.
func (c *Client) Delete(ctx context.Context, bucket, key string) *apierror.APIError {
	ctx, span := s3Tracer.Start(ctx, "s3.delete")
	defer span.End()

	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to delete object from S3."))
	}
	return nil
}

// Copy copies an object within a bucket. The destination inherits SSE-AES256 from the bucket default. Idempotent: re-copying overwrites the destination, so an attach retry is safe.
func (c *Client) Copy(ctx context.Context, bucket, srcKey, dstKey string) *apierror.APIError {
	ctx, span := s3Tracer.Start(ctx, "s3.copy")
	defer span.End()

	_, err := c.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:               aws.String(bucket),
		CopySource:           aws.String(bucket + "/" + srcKey),
		Key:                  aws.String(dstKey),
		ServerSideEncryption: s3types.ServerSideEncryptionAes256,
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to copy object in S3."))
	}
	return nil
}

// Get fetches an object's full bytes. The caller owns the returned slice; objects are expected to be modest (raw emails), so the whole body is read into memory.
func (c *Client) Get(ctx context.Context, bucket, key string) ([]byte, *apierror.APIError) {
	ctx, span := s3Tracer.Start(ctx, "s3.get")
	defer span.End()

	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to read object from S3."))
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to read object body from S3."))
	}
	return data, nil
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
		// S3 HeadObject returns a 404 as a smithy HTTP response error. It returns 403 instead of 404 when the caller lacks s3:ListBucket permission, so treat both as "not found".
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
