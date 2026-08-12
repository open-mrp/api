package service

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// resolveShipByCommitment works out when an order is due to ship, from the customer, its account group, or the account default, and says which of the three decided.
//
// The engine is pure and lives in internal/scheduling beside the rest of the planning maths; this only gathers its inputs. Both this and a make-to-order plan have to agree on what a promise means, and two implementations of one chain would eventually disagree.
func (s *salesOrderSvcImpl) resolveShipByCommitment(ctx context.Context, accountID string, order *domain.SalesOrder, issuedAt time.Time) (*domain.ShipByCommitment, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.resolve_ship_by_commitment")
	defer span.End()

	in := scheduling.LeadTimeInput{}

	// A buyer with no customer relation resolves no chain rather than failing: the account default still applies.
	chain, apiErr := s.repos.NewSalesOrderRepo().GetCustomerLeadTimeChain(ctx, accountID, order.BuyerAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if chain != nil {
		in.CustomerLeadTimeDays = chain.CustomerLeadTimeDays
		in.AccountGroupLeadTimeDays = chain.AccountGroupLeadTimeDays
	}

	// GetSettings is row-or-defaults, so an account that has never opened the settings page still commits to the solver's own default rather than to nothing.
	settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if settings != nil {
		days := int(settings.DefaultCustomerLeadTimeDays)
		in.AccountLeadTimeDays = &days
	}

	// Only a promised delivery date has transit subtracted from it, so the lookup is skipped entirely otherwise. This runs inside the issue transaction, and resolving a lane costs an address read the lead-time branch would throw away.
	var transit *scheduling.Transit
	if order.PromisedAt != nil {
		resolved, apiErr := s.resolveOrderTransit(ctx, accountID, order)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		transit = resolved
	}

	commitment, ok := scheduling.ResolveCommitment(issuedAt, order.PromisedAt, in, transit)
	if !ok {
		// Every level being unset is a misconfiguration rather than a normal state. Leaving the commitment unstamped is the honest outcome: an order with no ship-by date reads as uncommitted, where a fabricated one would read as a promise nobody made.
		return nil, nil
	}

	return &domain.ShipByCommitment{
		ShipByDate:        commitment.ShipByDate,
		LeadTimeDays:      commitment.LeadTimeDays,
		SourceCode:        commitment.Source,
		TransitDays:       commitment.TransitDays,
		TransitSourceCode: commitment.TransitSource,
	}, nil
}

// stampShipByCommitment resolves and writes an order's commitment. Called on issue, and again whenever a promised date moves on an order that is already issued.
func (s *salesOrderSvcImpl) stampShipByCommitment(ctx context.Context, accountID string, order *domain.SalesOrder, issuedAt time.Time) *apierror.APIError {
	commitment, apiErr := s.resolveShipByCommitment(ctx, accountID, order, issuedAt)
	if apiErr != nil {
		return apiErr
	}
	if commitment == nil {
		return nil
	}
	return s.repos.NewSalesOrderRepo().SetShipByCommitment(ctx, accountID, order.ID, commitment)
}
