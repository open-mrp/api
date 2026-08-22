package audit

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/constants"
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
