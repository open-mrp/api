package repository

import (
	"context"
	"io"
	"testing"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/cloud/s3"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
)

type memObjects struct {
	s3.StubClient
	objects map[string][]byte
	gets    int
}

func (m *memObjects) Upload(_ context.Context, _, key string, body io.Reader, _ string) *apierror.APIError {
	data, _ := io.ReadAll(body)
	m.objects[key] = data
	return nil
}

func (m *memObjects) Get(_ context.Context, _, key string) ([]byte, *apierror.APIError) {
	m.gets++
	data, ok := m.objects[key]
	if !ok {
		return nil, apierror.NewInternalError(nil, "missing")
	}
	return data, nil
}

func storedPayload(t *testing.T, payload contracts.RequestLogPayload) (*blobstore.Store, *memObjects, string) {
	t.Helper()
	objects := &memObjects{objects: map[string][]byte{}}
	store, err := blobstore.New(&blobstore.Config{Objects: objects, Bucket: "payloads"})
	if err != nil {
		t.Fatalf("blobstore.New: %v", err)
	}
	key := contracts.RequestLogPayloadKey("rlog_1")
	if apiErr := store.Put(context.Background(), key, payload); apiErr != nil {
		t.Fatalf("Put: %v", apiErr)
	}
	return store, objects, key
}

func TestLoadPayload_FillsBodiesFromObjectStore(t *testing.T) {
	query := `{"limit":"5"}`
	body := `{"name":"widget"}`
	response := `{"id":"itm_1"}`
	store, _, key := storedPayload(t, contracts.RequestLogPayload{QueryJSON: &query, RequestBodyJSON: &body, ResponseBodyJSON: &response})
	repo := &requestLogRepoImpl{payloads: store}

	read := &domain.RequestLogRead{}
	includes := []string{"query_params", "request_body", "response_body"}
	if apiErr := repo.loadPayload(context.Background(), read, &key, includes); apiErr != nil {
		t.Fatalf("loadPayload: %v", apiErr)
	}
	applyRequestedJSONIncludes(read, includes)

	if read.QueryJSON == nil || *read.QueryJSON != query {
		t.Errorf("query: got %v", read.QueryJSON)
	}
	if read.BodyJSON == nil || *read.BodyJSON != body {
		t.Errorf("request body: got %v", read.BodyJSON)
	}
	if read.ResponseJSON == nil || *read.ResponseJSON != response {
		t.Errorf("response body: got %v", read.ResponseJSON)
	}
}

// Inline logs store {} for an absent body, so an offloaded log must read the same way.
func TestLoadPayload_AbsentBodiesReadAsEmptyObject(t *testing.T) {
	stack := "goroutine 1"
	store, _, key := storedPayload(t, contracts.RequestLogPayload{StackTrace: &stack})
	repo := &requestLogRepoImpl{payloads: store}

	read := &domain.RequestLogRead{}
	if apiErr := repo.loadPayload(context.Background(), read, &key, []string{"request_body", "response_body"}); apiErr != nil {
		t.Fatalf("loadPayload: %v", apiErr)
	}
	if read.BodyJSON == nil || *read.BodyJSON != "{}" {
		t.Errorf("request body: got %v", read.BodyJSON)
	}
	if read.ResponseJSON == nil || *read.ResponseJSON != "{}" {
		t.Errorf("response body: got %v", read.ResponseJSON)
	}
	if read.QueryJSON != nil {
		t.Errorf("query: got %v, want nil", *read.QueryJSON)
	}
}

func TestLoadPayload_SkipsObjectStoreWithoutBodyIncludes(t *testing.T) {
	store, objects, key := storedPayload(t, contracts.RequestLogPayload{})
	repo := &requestLogRepoImpl{payloads: store}

	if apiErr := repo.loadPayload(context.Background(), &domain.RequestLogRead{}, &key, []string{"actor"}); apiErr != nil {
		t.Fatalf("loadPayload: %v", apiErr)
	}
	if objects.gets != 0 {
		t.Errorf("object store read %d times, want 0", objects.gets)
	}
}

func TestLoadPayload_LeavesInlineLogsAlone(t *testing.T) {
	repo := &requestLogRepoImpl{}
	inline := `{"a":1}`
	read := &domain.RequestLogRead{BodyJSON: &inline}

	if apiErr := repo.loadPayload(context.Background(), read, nil, []string{"request_body"}); apiErr != nil {
		t.Fatalf("loadPayload: %v", apiErr)
	}
	if read.BodyJSON != &inline {
		t.Error("inline body must be left in place")
	}
}

func TestLoadPayload_ErrorsWhenStoreMissing(t *testing.T) {
	repo := &requestLogRepoImpl{}
	key := "request-logs/rlog_1.json.gz"

	if apiErr := repo.loadPayload(context.Background(), &domain.RequestLogRead{}, &key, []string{"request_body"}); apiErr == nil {
		t.Fatal("expected an error when the log's payload cannot be read")
	}
}
