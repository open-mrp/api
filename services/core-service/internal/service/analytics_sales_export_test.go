package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
)

func salesExportLines() []domain.SalesEntry {
	issued := time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC)
	po := "PO-7"
	// The page query returns newest first; the export reads oldest first.
	return []domain.SalesEntry{
		{ID: "il_new", InvoiceDate: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC), InvoiceNumber: "42", SalesOrderNumber: "17",
			ProductSku: "SCK-001", QuantityInvoiced: 12, Unit: "pr", UnitPrice: 3, UnitCost: 1.5, TotalInvoiced: 36,
			CustomerName: "Acme", CustomerNumber: "C1", CustomerCreatedAt: issued},
		{ID: "il_old", IssuedAt: &issued, CustomerPO: &po, InvoiceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), InvoiceNumber: "41", SalesOrderNumber: "SO-9",
			ProductSku: "SCK-002", QuantityInvoiced: 5, Unit: "pr", UnitPrice: 4.25, UnitCost: 2, TotalInvoiced: 21.25,
			CustomerName: "Acme", CustomerNumber: "C1", CustomerCreatedAt: issued},
	}
}

func buildSalesExport(t *testing.T, hideCost bool) *domain.Export {
	t.Helper()
	ctrl := gomock.NewController(t)
	repos := factorymock.NewMockRepoFactory(ctrl)
	reports := repositorymock.NewMockSalesReportRepo(ctrl)
	repos.EXPECT().NewSalesReportRepo().Return(reports).AnyTimes()
	reports.EXPECT().GetLinePage(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p domain.ListSalesLinesParams) (*domain.SalesLinePage, *apierror.APIError) {
			assert.Equal(t, int32(domain.ExportRowLimit+1), p.Limit, "fetch one past the limit so an oversized export is refused")
			assert.Equal(t, "ac_1", p.AccountID)
			return &domain.SalesLinePage{Lines: salesExportLines()}, nil
		})

	svc := &analyticsSvcImpl{repos: repos}
	filters, err := json.Marshal(domain.ExportSalesLinesParams{HasWindow: true, HideCost: hideCost})
	require.NoError(t, err)
	export, apiErr := svc.BuildExportSalesLines(context.Background(), "ac_1", filters)
	require.Nil(t, apiErr)
	return export
}

func TestSalesExport_MatchesTheDashboardsFile(t *testing.T) {
	export := buildSalesExport(t, false)
	rows := exportedSheetRows(t, export, "Sales Data")
	require.Len(t, rows, 3, "a header and two lines")

	assert.Equal(t, []string{
		"Issued At", "Invoiced At", "Customer PO", "Invoice Number", "Sales Order Number", "Product Line", "SKU",
		"Product Description", "Qty Invoiced", "UOM", "Unit Price", "Unit Cost", "Total Invoiced", "Customer Group",
		"Customer Name", "Customer Number", "Customer Date Created", "Ship To State", "Ship To Country", "Ship To City",
		"Ship To Zipcode", "Sales Rep Name", "Parent Customer ID",
	}, rows[0])

	oldest := rows[1]
	assert.Equal(t, "03/01/2026", oldest[0], "issued at is a date cell in mm/dd/yyyy")
	assert.Equal(t, "03/02/2026", oldest[1], "oldest invoice first")
	assert.Equal(t, "PO-7", oldest[2])
	assert.Equal(t, "000041", oldest[3], "numeric record numbers are padded as the dashboard showed them")
	assert.Equal(t, "SO-9", oldest[4], "a prefixed number is left alone")
	assert.Equal(t, "5", oldest[8])
	assert.Equal(t, "4.25", oldest[10])
	assert.Equal(t, "2", oldest[11], "unit cost")
	assert.Equal(t, "21.25", oldest[12])

	newest := rows[2]
	assert.Equal(t, "", newest[0], "no issue date is a blank cell")
	assert.Equal(t, "000042", newest[3])
	assert.Equal(t, "000017", newest[4])
}

func TestSalesExport_SalesRepsGetNoCostColumn(t *testing.T) {
	export := buildSalesExport(t, true)
	rows := exportedSheetRows(t, export, "Sales Data")
	require.NotEmpty(t, rows)
	assert.NotContains(t, rows[0], "Unit Cost")
	assert.Equal(t, "Total Invoiced", rows[0][11], "the columns after it close up")
	assert.Equal(t, "21.25", rows[1][11])
}
