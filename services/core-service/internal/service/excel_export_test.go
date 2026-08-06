package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"
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
