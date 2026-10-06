package service

import (
	"context"
	"strings"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// salesReportAccess is who a sales report is for: the account it reads, and what the caller may see of it.
type salesReportAccess struct {
	accountID string
	// ownRepID is set for a sales rep, whose reports are forced to their own sales.
	ownRepID *string
	// includeCost is false for sales reps, who never see cost.
	includeCost bool
}

// salesReportAccessFor gates every sales report the same way the dashboard's analytics did: internal actors with invoice read access; sales reps see only their own sales and no cost.
func (s *analyticsSvcImpl) salesReportAccessFor(ctx context.Context) (*salesReportAccess, *apierror.APIError) {
	access, apiErr := s.orderReportAccessFor(ctx, types.PermissionDomainInvoices)
	if apiErr != nil {
		return nil, apiErr
	}

	ready, apiErr := s.reports().NewSalesReportRepo().FactsReady(ctx)
	if apiErr != nil {
		return nil, apiErr
	}
	if !ready {
		return nil, apierror.NewAPIError(apierror.ErrorCodeSvcUnavailable, apierror.ErrorTypeAPI,
			"Sales analytics are still being prepared. Try again in a few minutes.",
			"sales_line_fact backfill has not completed a pass")
	}
	return access, nil
}

// apply scopes a filter to the caller: their account, and their own sales when they are a sales rep.
func (a *salesReportAccess) apply(f *domain.SalesReportFilter) {
	f.AccountID = a.accountID
	if a.ownRepID != nil {
		f.SalesRepIDs = []string{*a.ownRepID}
	}
}

func (s *analyticsSvcImpl) AnalyzeSalesSummary(ctx context.Context, params domain.AnalyzeSalesSummaryParams) (*domain.SalesSummary, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_sales_summary")
	defer span.End()

	access, apiErr := s.salesReportAccessFor(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	access.apply(&params.SalesReportFilter)

	summary, apiErr := cachedReport(ctx, s.reportCache().salesSummary, analyticsReport{
		accountID: access.accountID,
		family:    analyticsFamilySales,
		method:    "sales_summary",
		params: struct {
			P    domain.AnalyzeSalesSummaryParams
			Cost bool
		}{params, access.includeCost},
		ttl:    s.reportCache().ttlForWindow(latestEnd(params.SalesReportFilter)),
		bypass: s.reportCache().salesSettling(ctx, access.accountID),
	}, func(ctx context.Context) (*domain.SalesSummary, *apierror.APIError) {
		return s.reports().NewSalesReportRepo().GetSummary(ctx, params, access.includeCost)
	})
	return summary, tracing.Trace(span, apiErr)
}

func (s *analyticsSvcImpl) AnalyzeSalesBreakdown(ctx context.Context, params domain.AnalyzeSalesBreakdownParams) (*domain.SalesBreakdown, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_sales_breakdown")
	defer span.End()

	if !params.GroupBy.IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("group_by must be one of: "+strings.Join(params.GroupBy.EnumValues(), ", ")+".", "group_by"))
	}
	access, apiErr := s.salesReportAccessFor(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	access.apply(&params.SalesReportFilter)

	breakdown, apiErr := cachedReport(ctx, s.reportCache().salesBreakdown, analyticsReport{
		accountID: access.accountID,
		family:    analyticsFamilySales,
		method:    "sales_breakdown",
		params: struct {
			P    domain.AnalyzeSalesBreakdownParams
			Cost bool
		}{params, access.includeCost},
		ttl:    s.reportCache().ttlForWindow(latestEnd(params.SalesReportFilter)),
		bypass: s.reportCache().salesSettling(ctx, access.accountID),
	}, func(ctx context.Context) (*domain.SalesBreakdown, *apierror.APIError) {
		return s.reports().NewSalesReportRepo().GetBreakdown(ctx, params, access.includeCost)
	})
	return breakdown, tracing.Trace(span, apiErr)
}

func (s *analyticsSvcImpl) AnalyzeSalesInvoices(ctx context.Context, params domain.AnalyzeSalesInvoicesParams) (*domain.SalesInvoicePage, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_sales_invoices")
	defer span.End()

	access, apiErr := s.salesReportAccessFor(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	access.apply(&params.SalesReportFilter)

	page, apiErr := s.reports().NewSalesReportRepo().GetInvoicePage(ctx, params)
	return page, tracing.Trace(span, apiErr)
}

func (s *analyticsSvcImpl) ListSalesLines(ctx context.Context, params domain.ListSalesLinesParams) (*domain.SalesLinePage, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.list_sales_lines")
	defer span.End()

	access, apiErr := s.salesReportAccessFor(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	access.apply(&params.SalesReportFilter)

	page, apiErr := s.reports().NewSalesReportRepo().GetLinePage(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Same redaction as AnalyzeSales and the dashboard's rows: a sales rep's lines hide the unit cost.
	if !access.includeCost {
		for i := range page.Lines {
			page.Lines[i].UnitCost = 0
		}
	}
	return page, nil
}

// latestEnd is the last instant a report reads, which decides how long it may be cached.
func latestEnd(f domain.SalesReportFilter) (end time.Time) {
	end = f.EndsAt
	if f.ComparisonEndsAt != nil && f.ComparisonEndsAt.After(end) {
		end = *f.ComparisonEndsAt
	}
	return end
}
