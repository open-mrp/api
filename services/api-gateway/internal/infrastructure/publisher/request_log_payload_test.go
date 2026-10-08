package publisher

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/cloud/s3"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/platform"
	"google.golang.org/protobuf/encoding/protojson"
)

type recordingObjects struct {
	s3.StubClient
	mu      sync.Mutex
	fail    bool
	objects map[string][]byte
}

func (r *recordingObjects) Upload(_ context.Context, _, key string, body io.Reader, _ string) *apierror.APIError {
	if r.fail {
		return apierror.NewInternalError(nil, "upload failed")
	}
	data, _ := io.ReadAll(body)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.objects[key] = data
	return nil
}

func newPayloadStore(t *testing.T, objects *recordingObjects) *blobstore.Store {
	t.Helper()
	store, err := blobstore.New(&blobstore.Config{Objects: objects, Bucket: "payloads"})
	if err != nil {
		t.Fatalf("blobstore.New: %v", err)
	}
	return store
}

func bodiedRequestLog(id string) *appctx.RequestLog {
	query := `{"limit":"5"}`
	body := `{"name":"widget"}`
	response := `{"id":"itm_1"}`
	stack := "goroutine 1 [running]"
	return &appctx.RequestLog{
		ID:              id,
		Method:          "POST",
		Host:            "api.example.com",
		Path:            "/v1/items",
		NormalizedRoute: "/v1/items",
		StatusCode:      200,
		OccurredAt:      time.Now().UTC(),
		QueryJSON:       &query,
		BodyJSON:        &body,
		ResponseJSON:    &response,
		StackTrace:      &stack,
	}
}

func publishedEvent(t *testing.T, repo *capturingOutboxRepo) *pb.RequestLog {
	t.Helper()
	input := repo.waitForCreate(t)
	var event pb.RequestLog
	if err := protojson.Unmarshal(input.Payload.Data, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	return &event
}

// The bodies are what make request_log and message_outbox large, so once stored they must leave the message entirely.
func TestPublisher_OffloadsPayloadToObjectStore(t *testing.T) {
	t.Parallel()
	repo := newCapturingOutboxRepo()
	objects := &recordingObjects{objects: map[string][]byte{}}
	pub := NewRequestLogOutboxPublisher(repo, nil, newPayloadStore(t, objects), "", constants.PlatformModeProduction)

	if err := pub.Create(context.Background(), bodiedRequestLog("rlog_offload")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	event := publishedEvent(t, repo)

	wantKey := contracts.RequestLogPayloadKey("rlog_offload")
	if event.GetPayloadKey() != wantKey {
		t.Fatalf("payload key: got %q want %q", event.GetPayloadKey(), wantKey)
	}
	if event.QueryJson != nil || event.BodyJson != nil || event.ResponseJson != nil || event.StackTrace != nil {
		t.Errorf("offloaded fields must not stay on the message: %+v", event)
	}

	objects.mu.Lock()
	stored := objects.objects[wantKey]
	objects.mu.Unlock()
	zr, err := gzip.NewReader(bytes.NewReader(stored))
	if err != nil {
		t.Fatalf("stored payload is not gzip: %v", err)
	}
	var payload contracts.RequestLogPayload
	if err := json.NewDecoder(zr).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.RequestBodyJSON == nil || *payload.RequestBodyJSON != `{"name":"widget"}` {
		t.Errorf("request body: got %v", payload.RequestBodyJSON)
	}
	if payload.ResponseBodyJSON == nil || *payload.ResponseBodyJSON != `{"id":"itm_1"}` {
		t.Errorf("response body: got %v", payload.ResponseBodyJSON)
	}
	if payload.QueryJSON == nil || payload.StackTrace == nil {
		t.Errorf("query and stack trace must be stored: %+v", payload)
	}
}

// A failed upload must not cost the log its bodies.
func TestPublisher_KeepsPayloadInlineWhenUploadFails(t *testing.T) {
	t.Parallel()
	repo := newCapturingOutboxRepo()
	objects := &recordingObjects{fail: true, objects: map[string][]byte{}}
	pub := NewRequestLogOutboxPublisher(repo, nil, newPayloadStore(t, objects), "", constants.PlatformModeProduction)

	if err := pub.Create(context.Background(), bodiedRequestLog("rlog_inline")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	event := publishedEvent(t, repo)

	if event.PayloadKey != nil {
		t.Errorf("payload key must be absent when the upload failed, got %q", event.GetPayloadKey())
	}
	if event.GetBodyJson() != `{"name":"widget"}` || event.GetResponseJson() != `{"id":"itm_1"}` {
		t.Errorf("bodies must stay inline: body=%q response=%q", event.GetBodyJson(), event.GetResponseJson())
	}
}

func TestPublisher_SkipsUploadWithoutPayload(t *testing.T) {
	t.Parallel()
	repo := newCapturingOutboxRepo()
	objects := &recordingObjects{objects: map[string][]byte{}}
	pub := NewRequestLogOutboxPublisher(repo, nil, newPayloadStore(t, objects), "", constants.PlatformModeProduction)

	rl := &appctx.RequestLog{ID: "rlog_empty", Method: "GET", Path: "/v1/ping", StatusCode: 200, OccurredAt: time.Now().UTC()}
	if err := pub.Create(context.Background(), rl); err != nil {
		t.Fatalf("Create: %v", err)
	}
	event := publishedEvent(t, repo)

	if event.PayloadKey != nil {
		t.Errorf("no payload key expected, got %q", event.GetPayloadKey())
	}
	objects.mu.Lock()
	defer objects.mu.Unlock()
	if len(objects.objects) != 0 {
		t.Errorf("no object expected, got %d", len(objects.objects))
	}
}
