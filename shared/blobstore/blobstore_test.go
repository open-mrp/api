package blobstore_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/cloud/s3"
	apierror "github.com/open-mrp/api/shared/errors"
)

type memObjects struct {
	s3.StubClient
	objects map[string][]byte
	types   map[string]string
}

func newMemObjects() *memObjects {
	return &memObjects{objects: map[string][]byte{}, types: map[string]string{}}
}

func (m *memObjects) Upload(_ context.Context, bucket, key string, body io.Reader, contentType string) *apierror.APIError {
	data, err := io.ReadAll(body)
	if err != nil {
		return apierror.NewInternalError(err, "read")
	}
	m.objects[bucket+"/"+key] = data
	m.types[bucket+"/"+key] = contentType
	return nil
}

func (m *memObjects) Get(_ context.Context, bucket, key string) ([]byte, *apierror.APIError) {
	data, ok := m.objects[bucket+"/"+key]
	if !ok {
		return nil, apierror.NewInternalError(nil, "missing")
	}
	return data, nil
}

func TestPutGetRoundTrip(t *testing.T) {
	t.Parallel()

	objects := newMemObjects()
	store, err := blobstore.New(&blobstore.Config{Objects: objects, Bucket: "payloads"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	type doc struct {
		Body string `json:"body"`
	}
	if apiErr := store.Put(context.Background(), "request-logs/rlog_1.json.gz", doc{Body: "hello"}); apiErr != nil {
		t.Fatalf("Put: %v", apiErr)
	}

	var got doc
	if apiErr := store.Get(context.Background(), "request-logs/rlog_1.json.gz", &got); apiErr != nil {
		t.Fatalf("Get: %v", apiErr)
	}
	if got.Body != "hello" {
		t.Errorf("body: got %q want %q", got.Body, "hello")
	}
}

// Objects are stored compressed: request and response bodies are repetitive JSON, and the bucket bills by the byte.
func TestPutStoresGzip(t *testing.T) {
	t.Parallel()

	objects := newMemObjects()
	store, _ := blobstore.New(&blobstore.Config{Objects: objects, Bucket: "payloads"})

	if apiErr := store.PutJSON(context.Background(), "k", []byte(`{"a":1}`)); apiErr != nil {
		t.Fatalf("PutJSON: %v", apiErr)
	}

	zr, err := gzip.NewReader(bytes.NewReader(objects.objects["payloads/k"]))
	if err != nil {
		t.Fatalf("stored object is not gzip: %v", err)
	}
	raw, _ := io.ReadAll(zr)
	if string(raw) != `{"a":1}` {
		t.Errorf("decompressed: got %s", raw)
	}
	if objects.types["payloads/k"] != "application/gzip" {
		t.Errorf("content type: got %q", objects.types["payloads/k"])
	}
}

func TestGetMissingObjectErrors(t *testing.T) {
	t.Parallel()

	store, _ := blobstore.New(&blobstore.Config{Objects: newMemObjects(), Bucket: "payloads"})
	if _, apiErr := store.GetJSON(context.Background(), "absent"); apiErr == nil {
		t.Fatal("expected an error for a missing object")
	}
}

func TestNewRequiresObjectsAndBucket(t *testing.T) {
	t.Parallel()

	if _, err := blobstore.New(&blobstore.Config{Bucket: "b"}); err == nil {
		t.Error("expected an error without objects")
	}
	if _, err := blobstore.New(&blobstore.Config{Objects: newMemObjects()}); err == nil {
		t.Error("expected an error without a bucket")
	}
}

func TestSpillKeepsSmallDocumentsInline(t *testing.T) {
	t.Parallel()

	objects := newMemObjects()
	store, _ := blobstore.New(&blobstore.Config{Objects: objects, Bucket: "payloads"})

	small := []byte(`{"a":1}`)
	inline, key, apiErr := store.Spill(context.Background(), "jobs/job_1.json.gz", small)
	if apiErr != nil {
		t.Fatalf("Spill: %v", apiErr)
	}
	if key != nil || string(inline) != string(small) {
		t.Errorf("small document must stay inline: key=%v inline=%s", key, inline)
	}
	if len(objects.objects) != 0 {
		t.Errorf("no object expected, got %d", len(objects.objects))
	}
}

func TestSpillStoresLargeDocumentsAndFillReadsThemBack(t *testing.T) {
	t.Parallel()

	objects := newMemObjects()
	store, _ := blobstore.New(&blobstore.Config{Objects: objects, Bucket: "payloads"})

	large := []byte(`"` + strings.Repeat("x", blobstore.InlineLimit) + `"`)
	inline, key, apiErr := store.Spill(context.Background(), "jobs/job_1.json.gz", large)
	if apiErr != nil {
		t.Fatalf("Spill: %v", apiErr)
	}
	if inline != nil || key == nil || *key != "jobs/job_1.json.gz" {
		t.Fatalf("large document must be stored: key=%v inline=%d bytes", key, len(inline))
	}

	got, apiErr := store.Fill(context.Background(), nil, key)
	if apiErr != nil {
		t.Fatalf("Fill: %v", apiErr)
	}
	if string(got) != string(large) {
		t.Error("Fill must return the stored document")
	}
}

// Local stacks run without a bucket; a nil Store must behave as "everything inline".
func TestNilStoreKeepsEverythingInline(t *testing.T) {
	t.Parallel()

	var store *blobstore.Store
	large := []byte(`"` + strings.Repeat("x", blobstore.InlineLimit) + `"`)
	inline, key, apiErr := store.Spill(context.Background(), "k", large)
	if apiErr != nil || key != nil || len(inline) != len(large) {
		t.Fatalf("nil store must keep the document inline: key=%v err=%v", key, apiErr)
	}

	got, apiErr := store.Fill(context.Background(), inline, nil)
	if apiErr != nil || string(got) != string(large) {
		t.Fatalf("Fill must return the inline copy: err=%v", apiErr)
	}

	stored := "k"
	if _, apiErr := store.Fill(context.Background(), nil, &stored); apiErr == nil {
		t.Fatal("a stored document cannot be read without a store")
	}
}
