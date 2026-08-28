package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/messaging"
)

// A no-op update (no changes, no metadata) must be skipped before any
// validation: nil outbox repo and missing identity would otherwise error.
func TestPublish_skipsNoOpUpdate(t *testing.T) {
	t.Parallel()

	apiErr := NewPublisher().Publish(context.Background(), nil, EventData{
		ServiceName:  "core-service",
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeUnitGroup,
		ResourceID:   "ug_123",
	})
	if apiErr != nil {
		t.Fatalf("expected no-op update to be skipped, got error: %v", apiErr)
	}
}

func TestPublish_doesNotSkipUpdateWithChanges(t *testing.T) {
	t.Parallel()

	apiErr := NewPublisher().Publish(context.Background(), nil, EventData{
		ServiceName:  "core-service",
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeUnitGroup,
		ResourceID:   "ug_123",
		Changes:      []FieldChange{NewFieldChange("name", "old", "new")},
	})
	if apiErr == nil {
		t.Fatal("expected update with changes to proceed past the skip (and fail on missing identity)")
	}
}

func TestPublish_doesNotSkipUpdateWithMetadata(t *testing.T) {
	t.Parallel()

	apiErr := NewPublisher().Publish(context.Background(), nil, EventData{
		ServiceName:  "core-service",
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeAccountUser,
		ResourceID:   "au_123",
		Metadata:     map[string]any{"password_rotated": true},
	})
	if apiErr == nil {
		t.Fatal("expected update with metadata to proceed past the skip (and fail on missing identity)")
	}
}

func TestPublish_doesNotSkipCreateOrDeleteWithoutChanges(t *testing.T) {
	t.Parallel()

	for _, action := range []constants.AuditAction{constants.AuditActionCreate, constants.AuditActionDelete} {
		apiErr := NewPublisher().Publish(context.Background(), nil, EventData{
			ServiceName:  "core-service",
			Action:       action,
			ResourceType: constants.ObjectTypeUnitGroup,
			ResourceID:   "ug_123",
		})
		if apiErr == nil {
			t.Fatalf("expected %s without changes to proceed past the skip (and fail on missing identity)", action)
		}
	}
}

// ---------------------------------------------------------------------------
// success path
// ---------------------------------------------------------------------------

type fakeOutboxRepo struct {
	created []messaging.OutboxMessageInput
	err     error
}

func (f *fakeOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	f.created = append(f.created, input)
	if f.err != nil {
		return 0, f.err
	}
	return int64(len(f.created)), nil
}

func auditIdentity() *types.Identity {
	accountID := "acct_1"
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_1",
			AccountID:    &accountID,
		},
	}
}

// authedContext carries everything Publish reads off the context on a normal mutation.
func authedContext() context.Context {
	ctx := appctx.WithIdentity(context.Background(), auditIdentity())
	return appctx.WithRequestID(ctx, "req_1")
}

func sampleEvent() EventData {
	return EventData{
		ServiceName:      "core-service",
		Action:           constants.AuditActionUpdate,
		ResourceType:     constants.ObjectTypeUnitGroup,
		ResourceID:       "ug_123",
		RootResourceType: constants.ObjectTypeUnit,
		RootResourceID:   "un_456",
		Changes:          []FieldChange{NewFieldChange("name", "old", "new")},
		Metadata:         map[string]any{"reason": "rename"},
	}
}

// publishOnce runs Publish against a fake outbox and returns the single message it enqueued.
func publishOnce(t *testing.T, ctx context.Context, data EventData) messaging.OutboxMessageInput {
	t.Helper()

	repo := &fakeOutboxRepo{}
	if apiErr := NewPublisher().Publish(ctx, repo, data); apiErr != nil {
		t.Fatalf("Publish: %v", apiErr)
	}
	if len(repo.created) != 1 {
		t.Fatalf("outbox writes: got %d want 1", len(repo.created))
	}
	return repo.created[0]
}

// decodePayload reads the audit payload back out of the enqueued AMQP message.
func decodePayload(t *testing.T, msg messaging.OutboxMessageInput) auditEventOutboxPayload {
	t.Helper()

	var payload auditEventOutboxPayload
	if err := json.Unmarshal(msg.Payload.Data, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

// The outbox envelope is the wire contract every audit event routes on; a wrong
// routing key or exchange delivers the event nowhere.
func TestPublish_writesOutboxEnvelope(t *testing.T) {
	t.Parallel()

	msg := publishOnce(t, authedContext(), sampleEvent())

	if msg.ServiceName != "core-service" {
		t.Errorf("ServiceName: got %q want core-service", msg.ServiceName)
	}
	if want := string(contracts.PlatformEventAuditLogged); msg.MessageType != want {
		t.Errorf("MessageType: got %q want %q", msg.MessageType, want)
	}
	if want := string(contracts.PlatformEventAuditLogged); msg.RoutingKey != want {
		t.Errorf("RoutingKey: got %q want %q", msg.RoutingKey, want)
	}
	if msg.Destination != messaging.ApplicationExchange {
		t.Errorf("Destination: got %q want %q", msg.Destination, messaging.ApplicationExchange)
	}
	if !strings.HasPrefix(msg.MessageID, string(id.MessageIDPrefix)+"_") {
		t.Errorf("MessageID: got %q want %s_ prefix", msg.MessageID, id.MessageIDPrefix)
	}
	// The envelope's ID and the AMQP message's ID must agree, or a causal chain cannot be traced back to the outbox row.
	if msg.Payload.MessageID != msg.MessageID {
		t.Errorf("Payload.MessageID: got %q want %q", msg.Payload.MessageID, msg.MessageID)
	}
	if msg.Payload.RequestID != "req_1" {
		t.Errorf("Payload.RequestID: got %q want req_1", msg.Payload.RequestID)
	}
	if msg.Payload.Identity == nil || !msg.Payload.Identity.IsTargetAccountSet() {
		t.Errorf("Payload.Identity: got %+v want the context identity", msg.Payload.Identity)
	}
}

func TestPublish_payloadCarriesEventData(t *testing.T) {
	t.Parallel()

	before := time.Now().UTC()
	msg := publishOnce(t, authedContext(), sampleEvent())
	after := time.Now().UTC()

	payload := decodePayload(t, msg)

	if !strings.HasPrefix(payload.TypeID, string(id.AuditEventIDPrefix)+"_") {
		t.Errorf("TypeID: got %q want %s_ prefix", payload.TypeID, id.AuditEventIDPrefix)
	}
	if payload.Action != constants.AuditActionUpdate {
		t.Errorf("Action: got %q want update", payload.Action)
	}
	if payload.ResourceType != constants.ObjectTypeUnitGroup || payload.ResourceID != "ug_123" {
		t.Errorf("resource: got %q/%q want unit_group/ug_123", payload.ResourceType, payload.ResourceID)
	}
	if payload.RootResourceType != constants.ObjectTypeUnit || payload.RootResourceID != "un_456" {
		t.Errorf("root resource: got %q/%q want unit/un_456", payload.RootResourceType, payload.RootResourceID)
	}
	if payload.ServiceName != "core-service" {
		t.Errorf("ServiceName: got %q want core-service", payload.ServiceName)
	}
	if len(payload.Changes) != 1 || payload.Changes[0].Field != "name" {
		t.Errorf("Changes: got %+v", payload.Changes)
	}
	if payload.Metadata["reason"] != "rename" {
		t.Errorf("Metadata: got %+v", payload.Metadata)
	}
	// audit_event stores a UTC timestamp; a local-zone value would shift every reading of the trail.
	if _, offset := payload.OccurredAt.Zone(); offset != 0 {
		t.Errorf("OccurredAt zone offset: got %d want 0", offset)
	}
	if payload.OccurredAt.Before(before) || payload.OccurredAt.After(after) {
		t.Errorf("OccurredAt %v outside [%v, %v]", payload.OccurredAt, before, after)
	}
}

// The producer's payload struct and the consumer's PublishedEvent are two
// independent definitions of one JSON contract: a field added to only one of
// them compiles, and silently drops on the way into audit_event.
func TestPublish_payloadRoundTripsAsPublishedEvent(t *testing.T) {
	t.Parallel()

	ctx := appctx.WithIdempotencyKeyID(authedContext(), "ik_1")
	ctx = appctx.WithPropagatedClientIP(ctx, "203.0.113.7")
	msg := publishOnce(t, ctx, sampleEvent())

	var consumed PublishedEvent
	if err := json.Unmarshal(msg.Payload.Data, &consumed); err != nil {
		t.Fatalf("unmarshal as PublishedEvent: %v", err)
	}

	produced := decodePayload(t, msg)
	if consumed.TypeID != produced.TypeID ||
		consumed.Action != produced.Action ||
		consumed.ResourceType != produced.ResourceType ||
		consumed.ResourceID != produced.ResourceID ||
		consumed.RootResourceType != produced.RootResourceType ||
		consumed.RootResourceID != produced.RootResourceID ||
		consumed.ServiceName != produced.ServiceName ||
		!consumed.OccurredAt.Equal(produced.OccurredAt) {
		t.Errorf("scalar fields diverged:\n produced %+v\n consumed %+v", produced, consumed)
	}
	if consumed.IdempotencyKeyID == nil || *consumed.IdempotencyKeyID != "ik_1" {
		t.Errorf("IdempotencyKeyID: got %v want ik_1", consumed.IdempotencyKeyID)
	}
	if consumed.SourceIP == nil || *consumed.SourceIP != "203.0.113.7" {
		t.Errorf("SourceIP: got %v want 203.0.113.7", consumed.SourceIP)
	}
	if len(consumed.Changes) != 1 || consumed.Changes[0].Field != "name" ||
		string(consumed.Changes[0].OldValue) != `"old"` || string(consumed.Changes[0].NewValue) != `"new"` {
		t.Errorf("Changes: got %+v", consumed.Changes)
	}
	if consumed.Metadata["reason"] != "rename" {
		t.Errorf("Metadata: got %+v", consumed.Metadata)
	}

	// Re-marshalling what the consumer parsed must reproduce the producer's bytes: a producer-only field would survive the comparison above but vanish here.
	reMarshalled, err := json.Marshal(consumed)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	var producedJSON, consumedJSON map[string]any
	if err := json.Unmarshal(msg.Payload.Data, &producedJSON); err != nil {
		t.Fatalf("unmarshal produced: %v", err)
	}
	if err := json.Unmarshal(reMarshalled, &consumedJSON); err != nil {
		t.Fatalf("unmarshal consumed: %v", err)
	}
	if !reflect.DeepEqual(producedJSON, consumedJSON) {
		t.Errorf("PublishedEvent does not round-trip the produced payload:\n produced %v\n consumed %v", producedJSON, consumedJSON)
	}
}

// The request log's IP is the real client address; the propagated one is only a
// fallback for a service reached over gRPC.
func TestPublish_sourceIPPrecedence(t *testing.T) {
	t.Parallel()

	ip := func(s string) *string { return &s }

	tests := []struct {
		name         string
		requestLog   *appctx.RequestLog
		propagatedIP string
		want         *string
	}{
		{
			name:         "request log wins over propagated",
			requestLog:   &appctx.RequestLog{ClientIPString: ip("198.51.100.1")},
			propagatedIP: "203.0.113.7",
			want:         ip("198.51.100.1"),
		},
		{
			name:         "nil client IP falls back to propagated",
			requestLog:   &appctx.RequestLog{},
			propagatedIP: "203.0.113.7",
			want:         ip("203.0.113.7"),
		},
		{
			name:         "empty client IP falls back to propagated",
			requestLog:   &appctx.RequestLog{ClientIPString: ip("")},
			propagatedIP: "203.0.113.7",
			want:         ip("203.0.113.7"),
		},
		{
			name:         "no request log uses propagated",
			propagatedIP: "203.0.113.7",
			want:         ip("203.0.113.7"),
		},
		{
			name:       "request log only",
			requestLog: &appctx.RequestLog{ClientIPString: ip("198.51.100.1")},
			want:       ip("198.51.100.1"),
		},
		{
			name: "neither source omits the field",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := authedContext()
			if tc.requestLog != nil {
				ctx = appctx.WithRequestLog(ctx, tc.requestLog)
			}
			if tc.propagatedIP != "" {
				ctx = appctx.WithPropagatedClientIP(ctx, tc.propagatedIP)
			}

			payload := decodePayload(t, publishOnce(t, ctx, sampleEvent()))

			switch {
			case tc.want == nil && payload.SourceIP != nil:
				t.Fatalf("SourceIP: got %q want nil", *payload.SourceIP)
			case tc.want != nil && payload.SourceIP == nil:
				t.Fatalf("SourceIP: got nil want %q", *tc.want)
			case tc.want != nil && *payload.SourceIP != *tc.want:
				t.Fatalf("SourceIP: got %q want %q", *payload.SourceIP, *tc.want)
			}
		})
	}
}

// An audit row has to say which idempotency key produced it, and must not
// invent one when the request carried none.
func TestPublish_idempotencyKeyID(t *testing.T) {
	t.Parallel()

	withKey := decodePayload(t, publishOnce(t, appctx.WithIdempotencyKeyID(authedContext(), "ik_1"), sampleEvent()))
	if withKey.IdempotencyKeyID == nil || *withKey.IdempotencyKeyID != "ik_1" {
		t.Errorf("IdempotencyKeyID: got %v want ik_1", withKey.IdempotencyKeyID)
	}

	withoutKey := decodePayload(t, publishOnce(t, authedContext(), sampleEvent()))
	if withoutKey.IdempotencyKeyID != nil {
		t.Errorf("IdempotencyKeyID: got %q want nil", *withoutKey.IdempotencyKeyID)
	}
	if bytes.Contains(publishOnce(t, authedContext(), sampleEvent()).Payload.Data, []byte("idempotency_key_id")) {
		t.Error("idempotency_key_id key should be omitted when the request carried no key")
	}
}

func TestPublish_requiresTargetAccount(t *testing.T) {
	t.Parallel()

	ctx := appctx.WithIdentity(context.Background(), &types.Identity{Type: types.IdentityActorTypeUser})
	repo := &fakeOutboxRepo{}

	apiErr := NewPublisher().Publish(ctx, repo, sampleEvent())
	if apiErr == nil {
		t.Fatal("expected an error for an identity with no target account")
	}
	if apiErr.Code != apierror.ErrorCodeInvalidCredentials {
		t.Errorf("Code: got %q want %q", apiErr.Code, apierror.ErrorCodeInvalidCredentials)
	}
	if len(repo.created) != 0 {
		t.Errorf("outbox writes: got %d want 0", len(repo.created))
	}
}

func TestPublish_nilOutboxRepoWithValidIdentity(t *testing.T) {
	t.Parallel()

	apiErr := NewPublisher().Publish(authedContext(), nil, sampleEvent())
	if apiErr == nil {
		t.Fatal("expected an error when no outbox repo is supplied")
	}
	if apiErr.Code != apierror.ErrorCodeInternalError {
		t.Errorf("Code: got %q want %q", apiErr.Code, apierror.ErrorCodeInternalError)
	}
}

// A failed outbox write must surface: swallowing it would let the business
// mutation commit with no audit trail.
func TestPublish_surfacesOutboxFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeOutboxRepo{err: errors.New("deadlock found")}

	apiErr := NewPublisher().Publish(authedContext(), repo, sampleEvent())
	if apiErr == nil {
		t.Fatal("expected the outbox failure to surface")
	}
	if !strings.Contains(apiErr.Error(), "deadlock found") {
		t.Errorf("error should wrap the repo failure, got %q", apiErr.Error())
	}
}
