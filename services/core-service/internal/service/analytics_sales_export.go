package service

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
	"github.com/open-mrp/api/shared/textutil"
)

// salesExportRow is one invoiced line, carrying whether the export it belongs to hides cost: the worker that renders the file knows the stored filters, not the caller.
type salesExportRow struct {
	entry    domain.SalesEntry
	hideCost bool
}

const salesExportDateFormat = "mm/dd/yyyy"

// salesExportColumns are the dashboard's former sales export, column for column.
var salesExportColumns = []excel.ColumnSpec{
	{Header: "Issued At", Key: "issuedAt", Width: 15, NumFmt: salesExportDateFormat},
	{Header: "Invoiced At", Key: "invoicedAt", Width: 15, NumFmt: salesExportDateFormat},
	{Header: "Customer PO", Key: "customerPO", Width: 18},
	{Header: "Invoice Number", Key: "invoiceNumber", Width: 18},
	{Header: "Sales Order Number", Key: "orderNumber", Width: 18},
	{Header: "Product Line", Key: "productLine", Width: 20},
	{Header: "SKU", Key: "productSku", Width: 18},
	{Header: "Product Description", Key: "productDescription", Width: 30},
	{Header: "Qty Invoiced", Key: "quantityInvoiced", Width: 14},
	{Header: "UOM", Key: "unit", Width: 10},
	{Header: "Unit Price", Key: "unitPrice", Width: 14},
	{Header: "Unit Cost", Key: "unitCost", Width: 14},
	{Header: "Total Invoiced", Key: "totalInvoiced", Width: 16},
	{Header: "Customer Group", Key: "customerGroupName", Width: 20},
	{Header: "Customer Name", Key: "customerName", Width: 25},
	{Header: "Customer Number", Key: "customerNumber", Width: 18},
	{Header: "Customer Date Created", Key: "customerCreatedAt", Width: 20, NumFmt: salesExportDateFormat},
	{Header: "Ship To State", Key: "shipToState", Width: 14},
	{Header: "Ship To Country", Key: "shipToCountry", Width: 14},
	{Header: "Ship To City", Key: "shipToCity", Width: 18},
	{Header: "Ship To Zipcode", Key: "shipToZipcode", Width: 14},
	{Header: "Sales Rep Name", Key: "salesRepUsername", Width: 18},
	{Header: "Parent Customer ID", Key: "parentCustomerID", Width: 20},
}

func (s *analyticsSvcImpl) salesExportSpec() exportSpec[salesExportRow, domain.ExportSalesLinesParams] {
	return exportSpec[salesExportRow, domain.ExportSalesLinesParams]{
		PermissionDomain: types.PermissionDomainInvoices,
		Name:             "Sales Data",
		Slug:             "sales_data",
		ResourceType:     constants.ObjectTypeInvoiceLine,

		ColumnsFor: func(rows []salesExportRow) []excel.ColumnSpec {
			if len(rows) == 0 || !rows[0].hideCost {
				return salesExportColumns
			}
			return slices.DeleteFunc(slices.Clone(salesExportColumns), func(c excel.ColumnSpec) bool { return c.Key == "unitCost" })
		},

		// Oldest first, as the dashboard's export was. One read of the page query, sized one past the
		// limit so an oversized export is refused rather than cut short.
		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportSalesLinesParams) ([]salesExportRow, *apierror.APIError) {
			filters.AccountID = accountID
			page, apiErr := repos.NewSalesReportRepo().GetLinePage(ctx, domain.ListSalesLinesParams{
				SalesReportFilter: filters.SalesReportFilter,
				HasWindow:         filters.HasWindow,
				Limit:             domain.ExportRowLimit + 1,
			})
			if apiErr != nil {
				return nil, apiErr
			}
			rows := make([]salesExportRow, len(page.Lines))
			for i, e := range page.Lines {
				rows[len(rows)-1-i] = salesExportRow{entry: e, hideCost: filters.HideCost}
			}
			return rows, nil
		},

		Project: func(row salesExportRow) excel.Row {
			e := row.entry
			cells := excel.Row{
				"issuedAt":           dateOrBlank(e.IssuedAt),
				"invoicedAt":         e.InvoiceDate,
				"customerPO":         excel.Str(e.CustomerPO),
				"invoiceNumber":      textutil.FormatRecordNumber(e.InvoiceNumber),
				"orderNumber":        textutil.FormatRecordNumber(e.SalesOrderNumber),
				"productLine":        excel.Str(e.ProductLine),
				"productSku":         e.ProductSku,
				"productDescription": excel.Str(e.ProductDescription),
				"quantityInvoiced":   e.QuantityInvoiced,
				"unit":               e.Unit,
				"unitPrice":          e.UnitPrice,
				"totalInvoiced":      e.TotalInvoiced,
				"customerGroupName":  excel.Str(e.CustomerGroupName),
				"customerName":       e.CustomerName,
				"customerNumber":     e.CustomerNumber,
				"customerCreatedAt":  e.CustomerCreatedAt,
				"shipToState":        excel.Str(e.ShipToState),
				"shipToCountry":      excel.Str(e.ShipToCountry),
				"shipToCity":         excel.Str(e.ShipToCity),
				"shipToZipcode":      excel.Str(e.ShipToPostalCode),
				"salesRepUsername":   excel.Str(e.SalesRepUsername),
				"parentCustomerID":   excel.Str(e.ParentCustomerID),
			}
			if !row.hideCost {
				cells["unitCost"] = e.UnitCost
			}
			return cells
		},
	}
}

// dateOrBlank is a real date cell, so the column's date format applies, or blank.
func dateOrBlank(t *time.Time) any {
	if t == nil {
		return ""
	}
	return *t
}

// ExportSalesLines scopes the export to what the caller may see before it is stored: the worker replays the stored filters with no caller to ask. A sales rep gets only their own sales and no cost column.
func (s *analyticsSvcImpl) ExportSalesLines(ctx context.Context, params domain.ExportSalesLinesParams) (*domain.Job, *apierror.APIError) {
	access, apiErr := s.salesReportAccessFor(ctx)
	if apiErr != nil {
		return nil, apiErr
	}
	access.apply(&params.SalesReportFilter)
	params.HideCost = !access.includeCost
	return enqueueExport(ctx, s.exportDeps(), s.salesExportSpec(), params)
}

// BuildExportSalesLines renders an accepted export, reading the replica like every report.
func (s *analyticsSvcImpl) BuildExportSalesLines(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.reports(), s.salesExportSpec())(ctx, accountID, filters)
}

func (s *analyticsSvcImpl) exportDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}
