package service

import (
	"testing"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionIntentsFor(t *testing.T) {
	creator := messaging.SubscriptionIntent{UserID: actActorUserID, Reason: "creator"}
	editor := messaging.SubscriptionIntent{UserID: actActorUserID, Reason: "editor"}
	rep := messaging.SubscriptionIntent{AccountUserID: actOwnerAccountUserID, Reason: "assignee"}
	cases := []struct {
		name  string
		event *domain.AuditEvent
		want  []messaging.SubscriptionIntent
	}{
		{"a person creates an order with a rep", orderEvent(actActorUserID, constants.AuditActionCreate, change("sales_rep_id", `null`, `"acus_owner"`)),
			[]messaging.SubscriptionIntent{creator, rep}},
		{"a person edits an order", orderEvent(actActorUserID, constants.AuditActionUpdate, change("note", `null`, `"x"`)),
			[]messaging.SubscriptionIntent{editor}},
		{"clearing the rep assigns nobody", orderEvent(actActorUserID, constants.AuditActionUpdate, change("sales_rep_id", `"acus_owner"`, `null`)),
			[]messaging.SubscriptionIntent{editor}},
		{"an API key creates a purchase order", &domain.AuditEvent{ID: "e", ActorID: "key_1", IdentityType: "api_key", AccountID: "acct_1",
			Action: constants.AuditActionCreate, ResourceType: constants.ObjectTypePurchaseOrder, ResourceID: "po_1"}, nil},
		{"a production run's responsible user", &domain.AuditEvent{ID: "e", ActorID: constants.SystemActorScheduler.ActorID(), IdentityType: "system", AccountID: "acct_1",
			Action: constants.AuditActionCreate, ResourceType: constants.ObjectTypeProductionRun, ResourceID: "pr_1",
			Changes: []domain.AuditFieldChange{change("responsible_user_id", `null`, `"acus_owner"`)}},
			[]messaging.SubscriptionIntent{rep}},
		{"a line isn't followable", childEvent(actActorUserID, constants.ObjectTypeSalesOrderLine, "sol_1", constants.AuditActionCreate), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, subscriptionIntentsFor(tc.event))
		})
	}
}

func TestSubscribeFromAuditEvent_EnqueuesSubscribeOnly(t *testing.T) {
	h := newActivityHarness(t, nil)
	ctx := namedActorCtx("acct_1", actActorUserID, "Blake Doe")
	event := &domain.AuditEvent{ID: "aevt_cust", ActorID: actActorUserID, IdentityType: "user", AccountID: "acct_1",
		Action: constants.AuditActionCreate, ResourceType: constants.ObjectTypeCustomer, ResourceID: "ac_cust"}

	assert.Empty(t, h.save(t, ctx, event))
	payloads := h.payloads(t)
	require.Len(t, payloads, 1)
	assert.Equal(t, "msg_sub_aevt_cust", h.outbox.inputs[0].MessageID)
	assert.Equal(t, &messaging.RecordRef{Type: "customer", ID: "ac_cust"}, payloads[0].Record)
	assert.Equal(t, []messaging.SubscriptionIntent{{UserID: actActorUserID, Reason: "creator"}}, payloads[0].Subscribe)
	assert.Nil(t, payloads[0].Audience)
}
