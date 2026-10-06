package service

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
	"github.com/open-mrp/api/shared/tracing"
)

// orderReportAccessFor gates an order-book report as the dashboard's analytics did: internal actors with read access to permissionDomain; a sales rep sees only their own orders and no cost.
func (s *analyticsSvcImpl) orderReportAccessFor(ctx context.Context, permissionDomain types.PermissionDomain) (*salesReportAccess, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := identity.CheckHasPermission(permissionDomain, types.ActionRead); apiErr != nil {
		return nil, apiErr
	}

	access := &salesReportAccess{accountID: identity.Target.AccountID, includeCost: !identity.IsSalesRep()}
	if identity.IsSalesRep() && identity.Actor != nil && identity.Actor.ID != "" {
		accountUser, apiErr := s.repos.NewAccountUserRepo().FindByAccountAndUserID(ctx, identity.Actor.ID, access.accountID)
		if apiErr != nil {
			return nil, apiErr
		}
		if accountUser != nil {
			access.ownRepID = &accountUser.ID
		}
	}
	return access, nil
}

// applyOpen scopes an open-order filter to the caller: their account, and their own orders when they are a sales rep.
func (a *salesReportAccess) applyOpen(f *domain.OpenOrderFilter) {
	f.AccountID = a.accountID
	if a.ownRepID != nil {
		f.SalesRepIDs = []string{*a.ownRepID}
	}
}

func (s *analyticsSvcImpl) AnalyzeOpenOrdersSummary(ctx context.Context, filter domain.OpenOrderFilter) (*domain.OpenOrdersSummary, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_open_orders_summary")
	defer span.End()

	access, apiErr := s.orderReportAccessFor(ctx, types.PermissionDomainInvoices)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	access.applyOpen(&filter)

	summary, apiErr := cachedReport(ctx, s.reportCache().openOrdersSummary, analyticsReport{
		accountID: access.accountID,
		family:    analyticsFamilySales,
		method:    "open_orders_summary",
		params:    filter,
		ttl:       s.reportCache().cfg.LiveTTL,
	}, func(ctx context.Context) (*domain.OpenOrdersSummary, *apierror.APIError) {
		return s.reports().NewAnalyticsRepo().GetOpenOrdersSummary(ctx, filter)
	})
	return summary, tracing.Trace(span, apiErr)
}

func (s *analyticsSvcImpl) AnalyzeOpenOrderProducts(ctx context.Context, params domain.AnalyzeOpenOrderProductsParams) (*domain.OpenOrderProductPage, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_open_order_products")
	defer span.End()

	access, apiErr := s.orderReportAccessFor(ctx, types.PermissionDomainInvoices)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	access.applyOpen(&params.OpenOrderFilter)
	params.Limit = clampOpenOrderLimit(params.Limit)

	page, apiErr := cachedReport(ctx, s.reportCache().openOrderProducts, analyticsReport{
		accountID: access.accountID,
		family:    analyticsFamilySales,
		method:    "open_order_products",
		params:    params,
		ttl:       s.reportCache().cfg.LiveTTL,
	}, func(ctx context.Context) (*domain.OpenOrderProductPage, *apierror.APIError) {
		page, apiErr := s.reports().NewAnalyticsRepo().GetOpenOrderProducts(ctx, params)
		if apiErr != nil {
			return nil, apiErr
		}
		ids := make([]string, len(page.Products))
		for i, p := range page.Products {
			ids[i] = p.UnitID
		}
		units, apiErr := s.unitsByID(ctx, access.accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for i := range page.Products {
			page.Products[i].Unit = units[page.Products[i].UnitID]
		}
		return page, nil
	})
	return page, tracing.Trace(span, apiErr)
}

// unitsByID loads the units a report's quantities are counted in. They travel with the report: a caller who may read it need not also be allowed to browse the account's units.
func (s *analyticsSvcImpl) unitsByID(ctx context.Context, accountID string, ids []string) (map[string]*domain.Unit, *apierror.APIError) {
	unique := dedupeStrings(ids)
	out := make(map[string]*domain.Unit, len(unique))
	if len(unique) == 0 {
		return out, nil
	}
	units, apiErr := s.reports().NewUnitRepo().GetByIDs(ctx, accountID, unique)
	if apiErr != nil {
		return nil, apiErr
	}
	for _, u := range units {
		out[u.ID] = u
	}
	return out, nil
}

func dedupeStrings(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok || id == "" {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *analyticsSvcImpl) ListOpenOrders(ctx context.Context, params domain.ListOpenOrdersParams) (*domain.OpenOrderPage, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.list_open_orders")
	defer span.End()

	access, apiErr := s.orderReportAccessFor(ctx, types.PermissionDomainInvoices)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	access.applyOpen(&params.OpenOrderFilter)
	params.Limit = clampOpenOrderLimit(params.Limit)

	page, apiErr := cachedReport(ctx, s.reportCache().openOrders, analyticsReport{
		accountID: access.accountID,
		family:    analyticsFamilySales,
		method:    "open_orders",
		params:    params,
		ttl:       s.reportCache().cfg.LiveTTL,
	}, func(ctx context.Context) (*domain.OpenOrderPage, *apierror.APIError) {
		return s.reports().NewAnalyticsRepo().ListOpenOrders(ctx, params)
	})
	return page, tracing.Trace(span, apiErr)
}

// ListOpenOrderLines returns every sale line of one sales order, open or not, as the dashboard's order detail did. A sales rep sees only their own orders.
func (s *analyticsSvcImpl) ListOpenOrderLines(ctx context.Context, orderID string) ([]domain.OpenOrderLine, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.list_open_order_lines")
	defer span.End()

	access, apiErr := s.orderReportAccessFor(ctx, types.PermissionDomainSalesOrders)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	lines, found, apiErr := s.reports().NewAnalyticsRepo().GetOpenOrderLines(ctx, access.accountID, orderID, access.ownRepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !found {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Sales order not found."))
	}
	ids := make([]string, len(lines))
	for i, l := range lines {
		ids[i] = l.UnitID
	}
	units, apiErr := s.unitsByID(ctx, access.accountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for i := range lines {
		lines[i].Unit = units[lines[i].UnitID]
	}
	return lines, nil
}

func clampOpenOrderLimit(limit int32) int32 {
	if limit <= 0 {
		return 10
	}
	return min(limit, 100)
}

// --- Export ---

// openOrderLineExportRow is one open line, carrying whether the export it belongs to hides cost: the worker that renders the file knows the stored filters, not the caller.
type openOrderLineExportRow struct {
	entry    domain.OrderEntry
	hideCost bool
}

// openOrderLineExportColumns are the dashboard's former products-on-order export, column for column.
var openOrderLineExportColumns = []excel.ColumnSpec{
	{Header: "Order Number", Key: "orderNumber", Width: 18},
	{Header: "Customer Number", Key: "customerNumber", Width: 18},
	{Header: "Customer", Key: "customerName", Width: 25},
	{Header: "Customer Group", Key: "customerGroupName", Width: 20},
	{Header: "Sales Rep", Key: "salesRepUsername", Width: 18},
	{Header: "SKU", Key: "productSku", Width: 18},
	{Header: "Description", Key: "productDescription", Width: 30},
	{Header: "Product Line", Key: "productLine", Width: 20},
	{Header: "Unit", Key: "unit", Width: 10},
	{Header: "Qty Ordered", Key: "quantityOrdered", Width: 14},
	{Header: "Qty Back Ordered", Key: "quantityBackOrdered", Width: 16},
	{Header: "Qty Invoiced", Key: "quantityInvoiced", Width: 14},
	{Header: "Unit Price", Key: "unitPrice", Width: 14},
	{Header: "Unit Cost", Key: "unitCost", Width: 14},
	{Header: "Total Ordered", Key: "totalOrdered", Width: 16},
	{Header: "Total Back Ordered", Key: "totalBackOrdered", Width: 18},
}

func (s *analyticsSvcImpl) openOrderLinesExportSpec() exportSpec[openOrderLineExportRow, domain.ExportOpenOrderLinesParams] {
	return exportSpec[openOrderLineExportRow, domain.ExportOpenOrderLinesParams]{
		PermissionDomain: types.PermissionDomainSalesOrders,
		Name:             "Products on Order",
		Slug:             "open_order_lines",
		ResourceType:     constants.ObjectTypeSalesOrderLine,

		ColumnsFor: func(rows []openOrderLineExportRow) []excel.ColumnSpec {
			if len(rows) == 0 || !rows[0].hideCost {
				return openOrderLineExportColumns
			}
			return slices.DeleteFunc(slices.Clone(openOrderLineExportColumns), func(c excel.ColumnSpec) bool { return c.Key == "unitCost" })
		},

		// Oldest order first, as the dashboard's export was, read one past the cap so an oversized export is refused rather than cut short.
		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportOpenOrderLinesParams) ([]openOrderLineExportRow, *apierror.APIError) {
			filters.AccountID = accountID
			entries, apiErr := repos.NewAnalyticsRepo().GetOpenOrderLineEntries(ctx, filters.OpenOrderFilter, domain.ExportRowLimit+1)
			if apiErr != nil {
				return nil, apiErr
			}
			rows := make([]openOrderLineExportRow, len(entries))
			for i, e := range entries {
				rows[i] = openOrderLineExportRow{entry: e, hideCost: filters.HideCost}
			}
			return rows, nil
		},

		Project: func(row openOrderLineExportRow) excel.Row {
			e := row.entry
			cells := excel.Row{
				"orderNumber":         e.OrderNumber,
				"customerNumber":      e.CustomerNumber,
				"customerName":        e.CustomerName,
				"customerGroupName":   excel.Str(e.CustomerGroupName),
				"salesRepUsername":    excel.Str(e.SalesRepUsername),
				"productSku":          e.ProductSku,
				"productDescription":  excel.Str(e.ProductDescription),
				"productLine":         excel.Str(e.ProductLine),
				"unit":                e.Unit,
				"quantityOrdered":     e.QuantityOrdered,
				"quantityBackOrdered": e.QuantityBackOrdered,
				"quantityInvoiced":    e.QuantityInvoiced,
				"unitPrice":           e.UnitPrice,
				"totalOrdered":        e.TotalOrdered,
				"totalBackOrdered":    e.TotalBackOrdered,
			}
			if !row.hideCost {
				cells["unitCost"] = e.UnitCost
			}
			return cells
		},
	}
}

// ExportOpenOrderLines scopes the export to what the caller may see before it is stored: the worker replays the stored filters with no caller to ask. A sales rep gets only their own orders and no cost column.
func (s *analyticsSvcImpl) ExportOpenOrderLines(ctx context.Context, params domain.ExportOpenOrderLinesParams) (*domain.Job, *apierror.APIError) {
	access, apiErr := s.orderReportAccessFor(ctx, types.PermissionDomainSalesOrders)
	if apiErr != nil {
		return nil, apiErr
	}
	access.applyOpen(&params.OpenOrderFilter)
	params.HideCost = !access.includeCost
	return enqueueExport(ctx, s.exportDeps(), s.openOrderLinesExportSpec(), params)
}

// BuildExportOpenOrderLines renders an accepted export, reading the replica like every report.
func (s *analyticsSvcImpl) BuildExportOpenOrderLines(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.reports(), s.openOrderLinesExportSpec())(ctx, accountID, filters)
}
