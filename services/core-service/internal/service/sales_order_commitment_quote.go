package service

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// QuoteSalesOrderCommitment previews the ship-by date a set of inputs would produce, without creating or changing anything.
//
// The whole point is that it runs the same resolution the issue path runs. An order-entry form that showed a date derived some other way would be worse than showing none: a rep would negotiate against a number the system then contradicts. Building a throwaway order and handing it to resolveShipByCommitment is what guarantees the two cannot disagree.
//
// Advisory rather than binding, for one honest reason: transit comes from a lane cache warmed asynchronously off order events, so a lane nobody has shipped yet has no estimate at preview time and the quote falls back to the service level's default, or to no transit at all. Rating the lane live would mean a fifteen-second carrier call on a keystroke, which is exactly what the cache exists to avoid.
func (s *salesOrderSvcImpl) QuoteSalesOrderCommitment(ctx context.Context, params domain.QuoteCommitmentParams) (*domain.ShipByCommitment, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.quote_commitment")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID

	if apiErr := validateCommitmentBasisExclusive(params.PromisedAt, params.LeadTimeOverrideDays, params.ShipByOverrideDate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	order, apiErr := s.commitmentQuoteSubject(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	issuedAt := time.Now()
	if params.IssuedAt != nil {
		issuedAt = *params.IssuedAt
	}

	commitment, apiErr := s.resolveShipByCommitment(ctx, params.AccountID, order, issuedAt, commitmentOptions{EstimateArrival: true})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// A nil commitment means no rule produced a date. The quote says so rather than inventing one, and the form shows the same nothing the order would have been issued with.
	if commitment == nil {
		return nil, nil
	}
	return commitment, nil
}

// commitmentQuoteSubject builds the order the preview is resolved against.
//
// A saved order supplies its own customer, address and carrier, and the request's bases override whatever it currently carries — that is how a detail page previews a change before saving it. An unsaved order has none of that, so the caller passes the parts directly and gets back the same resolution.
//
// Nothing here is written. The struct exists only to carry inputs into a resolution that reads.
func (s *salesOrderSvcImpl) commitmentQuoteSubject(ctx context.Context, params domain.QuoteCommitmentParams) (*domain.SalesOrder, *apierror.APIError) {
	order := &domain.SalesOrder{}

	if params.SalesOrderID != nil && *params.SalesOrderID != "" {
		existing, apiErr := s.repos.NewSalesOrderRepo().Get(ctx, params.AccountID, *params.SalesOrderID)
		if apiErr != nil {
			return nil, apiErr
		}
		order = existing
		// The saved bases are dropped rather than merged: a preview asks "what would this order commit to under these inputs", and leaving an old promised date in place would answer a question nobody asked.
		order.PromisedAt = nil
		order.LeadTimeOverrideDays = nil
		order.ShipByOverrideDate = nil
	} else {
		if params.BuyerAccountID != nil {
			order.BuyerAccountID = *params.BuyerAccountID
		}
		if params.ShipToAddressID != nil {
			order.ShippingAddressID = *params.ShipToAddressID
		}
		order.CarrierID = params.CarrierID
		order.ServiceLevelID = params.ServiceLevelID
	}

	order.PromisedAt = params.PromisedAt
	order.ShipByOverrideDate = params.ShipByOverrideDate
	if params.LeadTimeOverrideDays != nil {
		days := int(*params.LeadTimeOverrideDays)
		order.LeadTimeOverrideDays = &days
	}

	return order, nil
}
