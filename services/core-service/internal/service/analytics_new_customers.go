package service

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
)

// ListNewCustomers is the dashboard's new-customers table: internal actors with customer read access, as the
// legacy report required. It reads the buyer summaries, which answer only once their first pass has
// completed, so until then it is unavailable rather than wrong.
func (s *analyticsSvcImpl) ListNewCustomers(ctx context.Context, params domain.ListNewCustomersParams) (*domain.NewCustomerPage, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.list_new_customers")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID
	if params.EndsAt.Before(params.StartsAt) {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("ends_at", "ends_at must not be before starts_at."))
	}

	repo := s.reports().NewSalesReportRepo()
	ready, apiErr := repo.BuyerSummariesReady(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !ready {
		return nil, tracing.Trace(span, apierror.New(apierror.CodeServiceUnavailable, "The new customers report is still being prepared. Try again in a few minutes.", apierror.WithInternalMessage("sales_buyer_summary has not completed a pass")))
	}

	page, apiErr := cachedReport(ctx, s.reportCache().newCustomers, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilySales,
		method:    "new_customers",
		params:    params,
		// A customer's lifetime sales keep changing however old the window, so the live TTL always applies.
		ttl:    s.reportCache().ttlForWindow(time.Now().UTC()),
		bypass: s.reportCache().salesSettling(ctx, params.AccountID),
	}, func(ctx context.Context) (*domain.NewCustomerPage, *apierror.APIError) {
		return repo.GetNewCustomers(ctx, params)
	})
	return page, tracing.Trace(span, apiErr)
}
