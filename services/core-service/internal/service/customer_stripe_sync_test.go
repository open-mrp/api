package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
)

// capturingOutboxRepo records the outbox messages a mutation writes so a test can assert on the published command.
type capturingOutboxRepo struct {
	messages []messaging.OutboxMessageInput
}

func (r *capturingOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	r.messages = append(r.messages, input)
	return int64(len(r.messages)), nil
}

func TestStripeCustomerFieldsChanged(t *testing.T) {
	base := func() *domain.Customer {
		email := "billing@buyer.example.com"
		return &domain.Customer{Name: "Buyer Co", Number: "301064", Email: &email}
	}
	otherEmail := "new@buyer.example.com"

	tests := []struct {
		name    string
		old     *domain.Customer
		updated *domain.Customer
		want    bool
		wantWhy string
	}{
		{
			name:    "identical is unchanged",
			old:     base(),
			updated: base(),
			want:    false,
			wantWhy: "an edit to terms or carriers means nothing to Stripe",
		},
		{
			name:    "email change",
			old:     base(),
			updated: &domain.Customer{Name: "Buyer Co", Number: "301064", Email: &otherEmail},
			want:    true,
		},
		{
			name:    "name change",
			old:     base(),
			updated: &domain.Customer{Name: "Renamed Co", Number: "301064", Email: base().Email},
			want:    true,
		},
		{
			name:    "number change",
			old:     base(),
			updated: &domain.Customer{Name: "Buyer Co", Number: "999999", Email: base().Email},
			want:    true,
		},
		{
			name:    "email newly set from nil",
			old:     &domain.Customer{Name: "Buyer Co", Number: "301064"},
			updated: base(),
			want:    true,
			wantWhy: "this is the edit that makes the customer syncable at all",
		},
		{
			name:    "email cleared",
			old:     base(),
			updated: &domain.Customer{Name: "Buyer Co", Number: "301064"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, stripeCustomerFieldsChanged(tt.old, tt.updated), tt.wantWhy)
		})
	}
}

func TestEnqueueStripeCustomerSync_WritesOutboxCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	outbox := &capturingOutboxRepo{}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

	apiErr := enqueueStripeCustomerSync(context.Background(), repos, "ac_owner", "ac_buyer")
	require.Nil(t, apiErr)
	require.Len(t, outbox.messages, 1)

	msg := outbox.messages[0]
	require.Equal(t, string(contracts.CoreCmdSyncStripeCustomer), msg.MessageType)
	require.Equal(t, string(contracts.CoreCmdSyncStripeCustomer), msg.RoutingKey)
	require.Equal(t, messaging.ApplicationExchange, msg.Destination)

	// The payload carries identifiers only: the consumer re-reads the customer, so a
	// burst of edits converges on the current row instead of racing stale field values.
	var evt domain.SyncStripeCustomerEvent
	require.NoError(t, json.Unmarshal(msg.Payload.Data, &evt))
	require.Equal(t, "ac_owner", evt.OwnerAccountID)
	require.Equal(t, "ac_buyer", evt.CustomerAccountID)
}
