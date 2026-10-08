package repository

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/cloud/s3"
	apierror "github.com/open-mrp/api/shared/errors"
)

type memObjects struct {
	s3.StubClient
	objects map[string][]byte
}

func (m *memObjects) Upload(_ context.Context, _, key string, body io.Reader, _ string) *apierror.APIError {
	data, _ := io.ReadAll(body)
	m.objects[key] = data
	return nil
}

func (m *memObjects) Get(_ context.Context, _, key string) ([]byte, *apierror.APIError) {
	data, ok := m.objects[key]
	if !ok {
		return nil, apierror.NewInternalError(nil, "missing")
	}
	return data, nil
}

type failingObjects struct {
	s3.StubClient
	done chan struct{}
}

func (f *failingObjects) Upload(context.Context, string, string, io.Reader, string) *apierror.APIError {
	close(f.done)
	return apierror.NewInternalError(nil, "upload failed")
}

func newMemPayloads(t *testing.T) (*blobstore.Store, *memObjects) {
	t.Helper()
	objects := &memObjects{objects: map[string][]byte{}}
	store, err := blobstore.New(&blobstore.Config{Objects: objects, Bucket: "payloads"})
	if err != nil {
		t.Fatalf("blobstore.New: %v", err)
	}
	return store, objects
}

// A thousand-row bulk payload is what bloats the job table, so it must leave the row; the worker still reads it back whole.
func TestJobRepo_PutItems_StoresLargePayloadsInObjectStorage(t *testing.T) {
	t.Parallel()
	store, objects := newMemPayloads(t)
	repo := NewJobRepo(nil, store)

	items := json.RawMessage(`["` + strings.Repeat("x", blobstore.InlineLimit) + `"]`)
	inline, key, apiErr := repo.PutItems(context.Background(), "jb_large", items)
	if apiErr != nil {
		t.Fatalf("PutItems: %v", apiErr)
	}
	if inline != nil || key == nil || *key != "jobs/jb_large/items.json.gz" {
		t.Fatalf("expected the payload in object storage, got key=%v inline=%d bytes", key, len(inline))
	}
	if _, ok := objects.objects[*key]; !ok {
		t.Fatal("payload object was not written")
	}

	got, apiErr := repo.GetItems(context.Background(), &domain.Job{ID: "jb_large", JobItemsKey: key})
	if apiErr != nil {
		t.Fatalf("GetItems: %v", apiErr)
	}
	if string(got) != string(items) {
		t.Error("GetItems must return the stored payload")
	}
}

func TestJobRepo_PutItems_KeepsSmallPayloadsInline(t *testing.T) {
	t.Parallel()
	store, objects := newMemPayloads(t)
	repo := NewJobRepo(nil, store)

	items := json.RawMessage(`[{"name":"a"}]`)
	inline, key, apiErr := repo.PutItems(context.Background(), "jb_small", items)
	if apiErr != nil {
		t.Fatalf("PutItems: %v", apiErr)
	}
	if key != nil || string(inline) != string(items) {
		t.Fatalf("expected the payload inline, got key=%v", key)
	}
	if len(objects.objects) != 0 {
		t.Errorf("no object expected, got %d", len(objects.objects))
	}

	got, apiErr := repo.GetItems(context.Background(), &domain.Job{ID: "jb_small", JobItems: inline})
	if apiErr != nil || string(got) != string(items) {
		t.Fatalf("GetItems must return the inline payload: %v", apiErr)
	}
}
