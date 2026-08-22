package service

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

type commitmentOptions struct {
	EstimateArrival bool
}

// resolveShipByCommitment works out when an order is due to ship, from the customer, its parent account, its account group, or the account default, and says which of the four decided.
//
// The engine is pure and lives in internal/scheduling beside the rest of the planning maths; this only gathers its inputs. Both this and a make-to-order plan have to agree on what a promise means, and two implementations of one chain would eventually disagree.
func (s *salesOrderSvcImpl) resolveShipByCommitment(ctx context.Context, accountID string, order *domain.SalesOrder, issuedAt time.Time, opts commitmentOptions) (*domain.ShipByCommitment, *apierror.APIError) {
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
		in.ParentCustomerLeadTimeDays = chain.ParentCustomerLeadTimeDays
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

	basis := salesOrderCommitmentBasis(order)

	var transit *scheduling.Transit
	switch {
	case basis.PromisedAt != nil:
		dest, apiErr := s.resolveOrderAddress(ctx, s.repos.NewAddressRepo(), accountID, order.BuyerAccountID, order.ShippingAddressID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		transit, apiErr = s.resolveOrderTransit(ctx, accountID, order, dest)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

	case opts.EstimateArrival && order.ShippingAddressID != "" && order.ServiceLevelID != nil && *order.ServiceLevelID != "":
		if dest, apiErr := s.resolveOrderAddress(ctx, s.repos.NewAddressRepo(), accountID, order.BuyerAccountID, order.ShippingAddressID); apiErr == nil {
			if resolved, apiErr := s.resolveOrderTransit(ctx, accountID, order, dest); apiErr == nil {
				transit = resolved
			}
		}
	}

	// Anchored on whichever date the commitment is being worked back from, so the closure window covers the span the walk can actually reach.
	cals, apiErr := s.resolveCommitmentCalendars(ctx, accountID, order, commitmentAnchor(basis, in, issuedAt))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	commitment, ok := scheduling.ResolveCommitment(issuedAt, basis, in, transit, cals)
	if !ok {
		// Every level being unset is a misconfiguration rather than a normal state. Leaving the commitment unstamped is the honest outcome: an order with no ship-by date reads as uncommitted, where a fabricated one would read as a promise nobody made.
		return nil, nil
	}

	out := &domain.ShipByCommitment{
		ShipByDate:             commitment.ShipByDate,
		LeadTimeDays:           commitment.LeadTimeDays,
		SourceCode:             commitment.Source,
		TransitDays:            commitment.TransitDays,
		TransitSourceCode:      commitment.TransitSource,
		ShipByCutoffAt:         commitment.ShipByCutoffAt,
		CalendarAdjustmentDays: commitment.CalendarAdjustmentDays,
		Steps:                  commitmentSteps(commitment.Steps),
	}

	if opts.EstimateArrival && transit != nil {
		days := transit.Days
		out.TransitDays = &days
		out.TransitSourceCode = transit.Source
		if arrival, ok := scheduling.EstimateArrival(commitment.ShipByDate, transit, cals); ok {
			out.EstimatedDeliveryDate = &arrival
		}
	}

	return out, nil
}

// commitmentSteps carries the engine's derivation across the domain boundary, so a caller can explain a date without importing the planning engine.
func commitmentSteps(steps []scheduling.CommitmentStep) []domain.CommitmentStep {
	if len(steps) == 0 {
		return nil
	}
	out := make([]domain.CommitmentStep, 0, len(steps))
	for _, s := range steps {
		out = append(out, domain.CommitmentStep{Code: s.Code, Date: s.Date, DaysMoved: s.DaysMoved, Detail: s.Detail})
	}
	return out
}

// salesOrderCommitmentBasis reads whichever explicit basis the order carries.
//
// At most one can be set — the write path rejects more — so this reports them all and lets the engine apply its defensive ordering rather than deciding here.
func salesOrderCommitmentBasis(order *domain.SalesOrder) scheduling.CommitmentBasis {
	return scheduling.CommitmentBasis{
		PromisedAt:           order.PromisedAt,
		LeadTimeOverrideDays: order.LeadTimeOverrideDays,
		ShipByOverrideDate:   order.ShipByOverrideDate,
	}
}

// stampShipByCommitment resolves and writes an order's commitment. Called on issue, and again whenever a promised date moves on an order that is already issued.
func (s *salesOrderSvcImpl) stampShipByCommitment(ctx context.Context, accountID string, order *domain.SalesOrder, issuedAt time.Time) *apierror.APIError {
	commitment, apiErr := s.resolveShipByCommitment(ctx, accountID, order, issuedAt, commitmentOptions{})
	if apiErr != nil {
		return apiErr
	}
	if commitment == nil {
		return nil
	}
	return s.repos.NewSalesOrderRepo().SetShipByCommitment(ctx, accountID, order.ID, commitment)
}

// commitmentAnchor is the date the closure window is centred on: roughly where the commitment is expected to land, before any calendar has moved it.
//
// Every basis is projected rather than defaulting to the issue date, because a lead time of any length puts the ship-by date outside a window centred on issue — a forty-five day lead time would already miss it, and the closures the walk runs through would silently not exist. The estimate only has to be close: the window is months wide, and being a few days out costs nothing.
func commitmentAnchor(basis scheduling.CommitmentBasis, in scheduling.LeadTimeInput, issuedAt time.Time) time.Time {
	switch {
	case basis.ShipByOverrideDate != nil:
		return *basis.ShipByOverrideDate
	case basis.PromisedAt != nil:
		return *basis.PromisedAt
	case basis.LeadTimeOverrideDays != nil && *basis.LeadTimeOverrideDays >= 0:
		return issuedAt.AddDate(0, 0, *basis.LeadTimeOverrideDays)
	default:
		if days, _, ok := scheduling.ResolveLeadTime(in); ok {
			return issuedAt.AddDate(0, 0, days)
		}
		return issuedAt
	}
}
