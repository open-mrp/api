package service

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// GetCustomerLeadTime resolves the ship-by lead time a new order for this customer would be committed to, and names the rule that produced it.
//
// This is the same chain the issue path stamps onto an order, run without an order, so a rep can see the commitment before making it. It reads through the same pure resolver rather than restating the precedence, because a form that previews one number while issue stamps another is worse than no preview.
func (s *customerSvcImpl) GetCustomerLeadTime(ctx context.Context, customerAccountID string) (*domain.CustomerLeadTime, *apierror.APIError) {
	ctx, span := customerSvcTracer.Start(ctx, "service.customer.get_lead_time")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	chain, apiErr := s.repos.NewSalesOrderRepo().GetCustomerLeadTimeChain(ctx, accountID, customerAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Unlike the issue path, a missing relation is an error here rather than a fallback: the caller asked about a specific customer, and answering with the account default would describe a customer that does not exist.
	if chain == nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Customer not found."))
	}

	settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	in := scheduling.LeadTimeInput{
		CustomerLeadTimeDays:       chain.CustomerLeadTimeDays,
		ParentCustomerLeadTimeDays: chain.ParentCustomerLeadTimeDays,
		AccountGroupLeadTimeDays:   chain.AccountGroupLeadTimeDays,
	}
	if settings != nil {
		days := int(settings.DefaultCustomerLeadTimeDays)
		in.AccountLeadTimeDays = &days
	}

	days, source, ok := scheduling.ResolveLeadTime(in)
	if !ok {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("No lead time is configured at any level for this account."))
	}

	out := &domain.CustomerLeadTime{
		CustomerAccountID: customerAccountID,
		Days:              days,
		SourceCode:        source,
	}
	// Each is named only when it is what actually decided; a customer that has a parent or a group but overrides it did not inherit anything.
	switch source {
	case scheduling.LeadTimeSourceParentCustomer:
		out.ParentCustomerAccountID = chain.ParentCustomerAccountID
	case scheduling.LeadTimeSourceAccountGroup:
		out.AccountGroupID = chain.AccountGroupID
	}
	return out, nil
}
