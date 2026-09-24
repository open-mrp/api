package audit

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
)

func TestDecodeObservedEvent(t *testing.T) {
	t.Parallel()

	payload, _ := json.Marshal(auditEventOutboxPayload{
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeRole,
		ResourceID:   "role_123",
		Changes: []FieldChange{
			{Field: "permissions", OldValue: json.RawMessage(`[]`), NewValue: json.RawMessage(`["x"]`)},
		},
	})
	body, _ := json.Marshal(contracts.AmqpMessage{
		Identity: &types.Identity{Target: &types.IdentityTarget{AccountID: "acct_1"}},
		Data:     payload,
	})

	got, ok := decodeObservedEvent(body)
	if !ok {
		t.Fatal("expected event to decode")
	}
	want := ObservedEvent{
		AccountID:     "acct_1",
		Action:        constants.AuditActionUpdate,
		ResourceType:  constants.ObjectTypeRole,
		ResourceID:    "role_123",
		ChangedFields: []string{"permissions"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDecodeObservedEvent_ToleratesMissingIdentity(t *testing.T) {
	t.Parallel()

	payload, _ := json.Marshal(auditEventOutboxPayload{Action: constants.AuditActionDelete, ResourceType: constants.ObjectTypeAPIKey, ResourceID: "key_1"})
	body, _ := json.Marshal(contracts.AmqpMessage{Data: payload})

	got, ok := decodeObservedEvent(body)
	if !ok || got.AccountID != "" || got.ResourceID != "key_1" {
		t.Fatalf("got (%+v, %v)", got, ok)
	}
}

func TestDecodeObservedEvent_RejectsGarbage(t *testing.T) {
	t.Parallel()

	if _, ok := decodeObservedEvent([]byte("not json")); ok {
		t.Fatal("garbage decoded")
	}
}
