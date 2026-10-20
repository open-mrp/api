package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/stretchr/testify/require"
)

type auditOutboxRepo struct {
	created []messaging.OutboxMessageInput
}

func (r *auditOutboxRepo) Create(_ context.Context, in messaging.OutboxMessageInput) (int64, error) {
	r.created = append(r.created, in)
	return int64(len(r.created)), nil
}

// audits returns the audit events published, with the identity each is attributed to.
func (r *auditOutboxRepo) audits(t *testing.T) ([]audit.PublishedEvent, []*types.Identity) {
	t.Helper()
	var events []audit.PublishedEvent
	var identities []*types.Identity
	for _, msg := range r.created {
		if msg.RoutingKey != string(contracts.PlatformEventAuditLogged) {
			continue
		}
		var event audit.PublishedEvent
		require.NoError(t, json.Unmarshal(msg.Payload.Data, &event))
		events = append(events, event)
		identities = append(identities, msg.Payload.Identity)
	}
	return events, identities
}
