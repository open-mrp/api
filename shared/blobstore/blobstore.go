// Package blobstore keeps large JSON documents in object storage so the database row that owns one
// holds only its key. Documents are immutable and keyed by the owning record's id: a retry, a
// redelivery, or a recovery replay rewrites the same bytes under the same key, so every write is
// idempotent without coordination. Write the document before the row that points at it; a row that
// never lands leaves an unreferenced object the bucket's lifecycle rule expires.
package blobstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/open-mrp/api/shared/cloud/s3"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var tracer = tracing.GetTracer("shared.blobstore")

const contentType = "application/gzip"

// Config configures a Store.
type Config struct {
	// Objects (required) is the object store documents are written to and read from.
	Objects s3.ObjectStore
	// Bucket (required) is the bucket every document lives in.
	Bucket string
}

func (c *Config) validate() error {
	if c == nil {
		return fmt.Errorf("blobstore: config is nil")
	}
	if c.Objects == nil {
		return fmt.Errorf("blobstore: objects is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("blobstore: bucket is required")
	}
	return nil
}

// Store reads and writes gzipped JSON documents in one bucket.
type Store struct {
	objects s3.ObjectStore
	bucket  string
}

func New(cfg *Config) (*Store, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Store{objects: cfg.Objects, bucket: cfg.Bucket}, nil
}

// Open connects a Store to bucket in region. An empty bucket returns a nil Store, which callers
// treat as "keep documents inline": local stacks without a bucket keep working.
func Open(ctx context.Context, region, bucket string) (*Store, error) {
	if bucket == "" {
		slog.WarnContext(ctx, "blobstore: no bucket configured; large documents stay in the database")
		return nil, nil
	}
	client, apiErr := s3.NewClient(ctx, region)
	if apiErr != nil {
		return nil, fmt.Errorf("blobstore: %s", apiErr.PublicMessage)
	}
	return New(&Config{Objects: client, Bucket: bucket})
}

// InlineLimit is the largest document a row keeps itself. Most documents are well under it, so
// they cost no object-store round trip; the few large ones — bulk payloads, big responses — are
// what bloat the table and buffer pool, and those go to the bucket.
const InlineLimit = 16 << 10

// Spill decides where a JSON document lives. A document over InlineLimit is stored under key and
// its key returned for the row; anything smaller is returned for the row to keep inline. A nil
// Store keeps every document inline. Call it before the transaction that writes the row.
func (s *Store) Spill(ctx context.Context, key string, data []byte) (inline []byte, objectKey *string, apiErr *apierror.APIError) {
	if s == nil || len(data) <= InlineLimit {
		return data, nil, nil
	}
	if apiErr := s.PutJSON(ctx, key, data); apiErr != nil {
		return nil, nil, apiErr
	}
	return nil, &key, nil
}

// Fill returns a document Spill placed: the row's inline copy, or the object its key names.
func (s *Store) Fill(ctx context.Context, inline []byte, objectKey *string) ([]byte, *apierror.APIError) {
	if objectKey == nil {
		return inline, nil
	}
	if s == nil {
		return nil, apierror.NewInternalError(nil, "Document is stored in object storage, but no payloads bucket is configured.")
	}
	return s.GetJSON(ctx, *objectKey)
}

// Put stores v as gzipped JSON under key.
func (s *Store) Put(ctx context.Context, key string, v any) *apierror.APIError {
	data, err := json.Marshal(v)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to encode stored document.")
	}
	return s.PutJSON(ctx, key, data)
}

// PutJSON stores an already-encoded JSON document under key.
func (s *Store) PutJSON(ctx context.Context, key string, data []byte) *apierror.APIError {
	ctx, span := tracer.Start(ctx, "blobstore.put")
	defer span.End()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to compress stored document."))
	}
	if err := zw.Close(); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to compress stored document."))
	}

	// A seekable body lets the SDK checksum the upload up front and rewind it to retry.
	if apiErr := s.objects.Upload(ctx, s.bucket, key, bytes.NewReader(buf.Bytes()), contentType); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// Get decodes the document stored under key into v.
func (s *Store) Get(ctx context.Context, key string, v any) *apierror.APIError {
	data, apiErr := s.GetJSON(ctx, key)
	if apiErr != nil {
		return apiErr
	}
	if err := json.Unmarshal(data, v); err != nil {
		return apierror.NewInternalError(err, "Failed to decode stored document.")
	}
	return nil
}

// GetJSON returns the JSON document stored under key.
func (s *Store) GetJSON(ctx context.Context, key string) ([]byte, *apierror.APIError) {
	ctx, span := tracer.Start(ctx, "blobstore.get")
	defer span.End()

	compressed, apiErr := s.objects.Get(ctx, s.bucket, key)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	zr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decompress stored document."))
	}
	defer zr.Close()

	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decompress stored document."))
	}
	return data, nil
}
