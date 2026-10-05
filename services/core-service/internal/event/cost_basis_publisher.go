package event

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// Reasons an item's cost basis moved, carried on the event for whoever reads the outbox later.
const (
	CostBasisProductionStepCreated   = "production_step_created"
	CostBasisProductionStepUpdated   = "production_step_updated"
	CostBasisProductionStepDeleted   = "production_step_deleted"
	CostBasisProductionStepsImported = "production_steps_imported"
	CostBasisConsumptionCreated      = "consumption_created"
	CostBasisConsumptionUpdated      = "consumption_updated"
	CostBasisConsumptionDeleted      = "consumption_deleted"
	CostBasisProductionUpdated       = "production_updated"
	CostBasisRateUpdated             = "rate_updated"
	CostBasisUnitCostUpdated         = "unit_cost_updated"
)

// PublishItemCostBasisChanged writes, on the outbox of the transaction that made the change, the event
// that something itemID's cost is derived from moved. Costing recomputes the item and everything built
// from it; the event names only where the change happened.
func PublishItemCostBasisChanged(ctx context.Context, repos domain.RepoFactory, accountID, itemID, reason string) *apierror.APIError {
	if itemID == "" {
		return nil
	}

	payload, err := json.Marshal(domain.ItemCostBasisChangedEvent{AccountID: accountID, ItemID: itemID, Reason: reason})
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal item cost basis changed event.")
	}

	msg := contracts.AmqpMessage{Data: payload}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	if _, err := repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreEventItemCostBasisChanged),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreEventItemCostBasisChanged),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for item cost basis changed event.")
	}
	return nil
}
