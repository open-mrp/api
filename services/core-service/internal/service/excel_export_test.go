package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
)

// reopens an export so assertions read what a spreadsheet reader would, rather
// than what the builder was handed
func openExportFile(t *testing.T, export *domain.Export) *excelize.File {
	t.Helper()
	require.NotNil(t, export, "the export is nil, so there is no workbook to read")
	f, err := excelize.OpenReader(bytes.NewReader(export.Body))
	require.NoError(t, err, "the export body must be a readable xlsx file")
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// reads one sheet out of an export. Trailing blank cells are trimmed on read, so
// a row ends at its last filled column.
func exportedSheetRows(t *testing.T, export *domain.Export, sheet string) [][]string {
	t.Helper()
	rows, err := openExportFile(t, export).GetRows(sheet)
	require.NoError(t, err, "no %q sheet in the export", sheet)
	return rows
}

// builds a spec over plain ints, enough for the row-count guard to act on
func countingExportSpec(rowCount int) exportSpec[int, struct{}] {
	return exportSpec[int, struct{}]{
		Name:    "Widgets",
		Slug:    "widgets",
		Columns: []excel.ColumnSpec{{Header: "N", Key: "n"}},
		Fetch: func(context.Context, domain.RepoFactory, string, struct{}) ([]int, *apierror.APIError) {
			return make([]int, rowCount), nil
		},
		Project: func(row int) excel.Row { return excel.Row{"n": row} },
	}
}

// Fetch reads one row past the cap, so an account too large to hold in memory must fail
// the job rather than render a silently truncated sheet.
func TestBuildExport_RejectsAnExportOverTheRowLimit(t *testing.T) {
	export, apiErr := buildExport(context.Background(), nil, countingExportSpec(domain.ExportRowLimit+1), "ac_1", struct{}{})

	require.Nil(t, export)
	require.NotNil(t, apiErr)
	require.False(t, apiErr.IsTransient, "an oversized export is deterministic: retrying it cannot help")
}

func TestBuildExport_RendersAnExportUnderTheRowLimit(t *testing.T) {
	export, apiErr := buildExport(context.Background(), nil, countingExportSpec(2), "ac_1", struct{}{})

	require.Nil(t, apiErr)
	require.NotNil(t, export)
	require.Equal(t, int32(2), export.RowCount)
}

// A cost column reaches only a requester who may see costs: an admin or a role with costs:read. Anyone else, a customer or supplier exporting through the portal included, gets the file without it.
func TestBuildExport_LeavesCostColumnsOutForARequesterWithoutCostsRead(t *testing.T) {
	t.Parallel()

	spec := exportSpec[int, struct{}]{
		Name:        "Widgets",
		Slug:        "widgets",
		Columns:     []excel.ColumnSpec{{Header: "SKU", Key: "sku"}, {Header: "Unit Cost", Key: "unit_cost"}},
		CostColumns: []string{"unit_cost"},
		Fetch: func(context.Context, domain.RepoFactory, string, struct{}) ([]int, *apierror.APIError) {
			return []int{1}, nil
		},
		Project: func(int) excel.Row { return excel.Row{"sku": "W-1", "unit_cost": "4.25"} },
	}
	customer := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeAPIKey,
		Target: &types.IdentityTarget{AccountID: "ac_seller"},
		Actor:  &types.IdentityActor{RelationType: types.IdentityRelationTypeCustomer, ID: "apky_buyer", AccountID: new("ac_buyer"), Permissions: map[string]bool{"costs:read": true}},
	})

	tests := []struct {
		name string
		ctx  context.Context
		want [][]string
	}{
		{"admin", salesCtx("ac_seller", string(constants.RoleTypeAdmin), nil), [][]string{{"SKU", "Unit Cost"}, {"W-1", "4.25"}}},
		{"role with costs:read", salesCtx("ac_seller", string(constants.RoleTypeCustom), map[string]bool{"costs:read": true}), [][]string{{"SKU", "Unit Cost"}, {"W-1", "4.25"}}},
		{"role without costs:read", salesCtx("ac_seller", string(constants.RoleTypeCustom), map[string]bool{"products:read": true}), [][]string{{"SKU"}, {"W-1"}}},
		{"customer portal actor", customer, [][]string{{"SKU"}, {"W-1"}}},
		{"no identity", context.Background(), [][]string{{"SKU"}, {"W-1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			export, apiErr := buildExport(tt.ctx, nil, spec, "ac_seller", struct{}{})
			require.Nil(t, apiErr)
			require.Equal(t, tt.want, exportedSheetRows(t, export, "Widgets"))
		})
	}
}
