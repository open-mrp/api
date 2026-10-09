package publisher

import (
	"context"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
)

func (r *recordingObjects) Get(_ context.Context, _, key string) ([]byte, *apierror.APIError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, ok := r.objects[key]
	if !ok {
		return nil, apierror.NewInternalError(nil, "missing")
	}
	return data, nil
}

// A body that could not go to object storage rides the outbox message instead; past the message
// cap it is replaced by a marker so one large body cannot bloat message_outbox.
func TestPublisher_CapsBodiesKeptOnTheMessage(t *testing.T) {
	t.Parallel()
	repo := newCapturingOutboxRepo()
	objects := &recordingObjects{fail: true, objects: map[string][]byte{}}
	pub := NewRequestLogOutboxPublisher(repo, nil, newPayloadStore(t, objects), "", constants.PlatformModeDevelopment)

	rl := bodiedRequestLog("rlog_capped")
	large := `"` + strings.Repeat("x", appctx.MaxMessageBodyBytes) + `"`
	rl.ResponseJSON = &large

	if err := pub.Create(context.Background(), rl); err != nil {
		t.Fatalf("Create: %v", err)
	}
	event := publishedEvent(t, repo)

	if !strings.HasPrefix(event.GetResponseJson(), `{"_truncated":true`) {
		t.Errorf("an oversized body must be replaced by a marker, got %d bytes", len(event.GetResponseJson()))
	}
	if event.GetBodyJson() != `{"name":"widget"}` {
		t.Errorf("a small body stays whole, got %q", event.GetBodyJson())
	}
}

// A body that did reach object storage is kept whole there, however far past the message cap.
func TestPublisher_StoresLargeBodiesWhole(t *testing.T) {
	t.Parallel()
	repo := newCapturingOutboxRepo()
	objects := &recordingObjects{objects: map[string][]byte{}}
	store := newPayloadStore(t, objects)
	pub := NewRequestLogOutboxPublisher(repo, nil, store, "", constants.PlatformModeDevelopment)

	rl := bodiedRequestLog("rlog_large")
	large := `"` + strings.Repeat("x", appctx.MaxMessageBodyBytes*4) + `"`
	rl.ResponseJSON = &large

	if err := pub.Create(context.Background(), rl); err != nil {
		t.Fatalf("Create: %v", err)
	}
	publishedEvent(t, repo)

	var payload contracts.RequestLogPayload
	if apiErr := store.Get(context.Background(), contracts.RequestLogPayloadKey("rlog_large"), &payload); apiErr != nil {
		t.Fatalf("stored payload: %v", apiErr)
	}
	if payload.ResponseBodyJSON == nil || len(*payload.ResponseBodyJSON) != len(large) {
		t.Errorf("stored response body was cut: got %d bytes, want %d", len(*payload.ResponseBodyJSON), len(large))
	}
}
