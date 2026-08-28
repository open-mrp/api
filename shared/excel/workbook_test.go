package excel

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// reopens built bytes so assertions read what Excel would, not what was intended
func reopen(t *testing.T, data []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func stationSheet() Sheet {
	return Sheet{
		Name: "Scanning Stations",
		Columns: []ColumnSpec{
			{Header: "ID", Key: "id", Width: 24},
			{Header: "Name", Key: "name", Width: 28},
			{Header: "Type", Key: "type", Width: 18},
			{Header: "Department", Key: "department", Width: 24, Note: "Must already exist."},
		},
		Rows: []Row{
			{"id": "ss_1", "name": "Packaging Line 1", "type": "init_batch", "department": "Cutting"},
			{"id": "ss_2", "name": "Merge Station", "type": "merge_batch", "department": "Sewing"},
		},
	}
}

func TestBuild_WritesHeadersRowsAndWidths(t *testing.T) {
	data, err := Build(Spec{Sheets: []Sheet{stationSheet()}})
	require.NoError(t, err)

	f := reopen(t, data)

	rows, err := f.GetRows("Scanning Stations")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, []string{"ID", "Name", "Type", "Department"}, rows[0])
	assert.Equal(t, []string{"ss_1", "Packaging Line 1", "init_batch", "Cutting"}, rows[1])
	assert.Equal(t, []string{"ss_2", "Merge Station", "merge_batch", "Sewing"}, rows[2])

	width, err := f.GetColWidth("Scanning Stations", "B")
	require.NoError(t, err)
	assert.InDelta(t, 28.0, width, 0.01)
}

func TestBuild_HeaderRowIsBold(t *testing.T) {
	data, err := Build(Spec{Sheets: []Sheet{stationSheet()}})
	require.NoError(t, err)

	f := reopen(t, data)
	styleID, err := f.GetCellStyle("Scanning Stations", "A1")
	require.NoError(t, err)
	style, err := f.GetStyle(styleID)
	require.NoError(t, err)
	require.NotNil(t, style.Font)
	assert.True(t, style.Font.Bold)
}

// a cell is placed by its column's key, never by a position the caller guessed
func TestBuild_PlacesCellsByColumnKey(t *testing.T) {
	sheet := Sheet{
		Name: "Keys",
		Columns: []ColumnSpec{
			{Header: "First", Key: "a", Width: 10},
			{Header: "Second", Key: "b", Width: 10},
			{Header: "Third", Key: "c", Width: 10},
		},
		// Deliberately out of column order, and missing "b".
		Rows: []Row{{"c": "third", "a": "first"}},
	}

	data, err := Build(Spec{Sheets: []Sheet{sheet}})
	require.NoError(t, err)

	f := reopen(t, data)
	for cell, want := range map[string]string{"A2": "first", "B2": "", "C2": "third"} {
		got, err := f.GetCellValue("Keys", cell)
		require.NoError(t, err)
		assert.Equal(t, want, got, "cell %s", cell)
	}
}

func TestBuild_HeaderNoteBecomesAComment(t *testing.T) {
	data, err := Build(Spec{Sheets: []Sheet{stationSheet()}})
	require.NoError(t, err)

	f := reopen(t, data)
	comments, err := f.GetComments("Scanning Stations")
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "D1", comments[0].Cell)
	assert.Contains(t, comments[0].Text, "Must already exist.")
}

func TestBuild_AppliesNumberFormatToDataCells(t *testing.T) {
	sheet := Sheet{
		Name:    "Prices",
		Columns: []ColumnSpec{{Header: "Unit Price", Key: "price", Width: 14, NumFmt: "#,##0.00"}},
		Rows:    []Row{{"price": 12.5}},
	}

	data, err := Build(Spec{Sheets: []Sheet{sheet}})
	require.NoError(t, err)

	f := reopen(t, data)
	styleID, err := f.GetCellStyle("Prices", "A2")
	require.NoError(t, err)
	style, err := f.GetStyle(styleID)
	require.NoError(t, err)
	require.NotNil(t, style.CustomNumFmt)
	assert.Equal(t, "#,##0.00", *style.CustomNumFmt)
}

// an empty result set still has to open, carrying just its headers
func TestBuild_NoRowsYieldsHeaderOnlySheet(t *testing.T) {
	sheet := stationSheet()
	sheet.Rows = nil

	data, err := Build(Spec{Sheets: []Sheet{sheet}})
	require.NoError(t, err)

	rows, err := reopen(t, data).GetRows("Scanning Stations")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, []string{"ID", "Name", "Type", "Department"}, rows[0])
}

func TestBuild_MultipleSheets(t *testing.T) {
	data, err := Build(Spec{Sheets: []Sheet{
		{Name: "product", Columns: []ColumnSpec{{Header: "SKU", Key: "sku"}}, Rows: []Row{{"sku": "P-1"}}},
		{Name: "material", Columns: []ColumnSpec{{Header: "SKU", Key: "sku"}}, Rows: []Row{{"sku": "M-1"}}},
	}})
	require.NoError(t, err)

	f := reopen(t, data)
	assert.Equal(t, []string{"product", "material"}, f.GetSheetList())
	assert.NotContains(t, f.GetSheetList(), "Sheet1")
}

func TestBuild_RejectsSpecsExcelCannotRepresent(t *testing.T) {
	tests := []struct {
		name    string
		spec    Spec
		wantErr string
	}{
		{
			name:    "no sheets",
			spec:    Spec{},
			wantErr: "at least one sheet",
		},
		{
			name:    "no columns",
			spec:    Spec{Sheets: []Sheet{{Name: "Empty"}}},
			wantErr: "has no columns",
		},
		{
			name: "duplicate column key",
			spec: Spec{Sheets: []Sheet{{Name: "Dupe", Columns: []ColumnSpec{
				{Header: "A", Key: "k"}, {Header: "B", Key: "k"},
			}}}},
			wantErr: "duplicate column key",
		},
		{
			name:    "column without a key",
			spec:    Spec{Sheets: []Sheet{{Name: "NoKey", Columns: []ColumnSpec{{Header: "A"}}}}},
			wantErr: "has no key",
		},
		{
			// "Sheet1" is the name excelize gives the default sheet, which Build
			// deletes once the spec's own sheets exist. A spec that claims the
			// name loses the sheet, so it must not build.
			name: "sheet named Sheet1",
			spec: Spec{Sheets: []Sheet{
				{Name: "Sheet1", Columns: []ColumnSpec{{Header: "A", Key: "a"}}},
				{Name: "Other", Columns: []ColumnSpec{{Header: "A", Key: "a"}}},
			}},
			wantErr: "Sheet1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Build(tc.spec)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// a spec must serialize identically every time, or a caller cannot cache or
// compare the bytes
func TestBuild_IsDeterministic(t *testing.T) {
	first, err := Build(Spec{Sheets: []Sheet{stationSheet()}})
	require.NoError(t, err)
	second, err := Build(Spec{Sheets: []Sheet{stationSheet()}})
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestFilename(t *testing.T) {
	at := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	// One separator throughout, so the name needs no escaping in a URL path.
	assert.Equal(t, "scanning_stations_export_07-30-2026.xlsx", Filename("scanning_stations", at))
}

func TestStr(t *testing.T) {
	value := "present"
	assert.Equal(t, "present", Str(&value))
	assert.Equal(t, "", Str(nil))
}

// a key a row omits and a key a row sets to nil are the same absent value, and
// Excel spells both of them as an empty cell
func TestBuild_NilCellIsBlank(t *testing.T) {
	t.Parallel()

	sheet := Sheet{
		Name:    "Nils",
		Columns: []ColumnSpec{{Header: "A", Key: "a"}, {Header: "B", Key: "b"}},
		Rows:    []Row{{"a": nil, "b": "set"}},
	}

	data, err := Build(Spec{Sheets: []Sheet{sheet}})
	require.NoError(t, err)

	f := reopen(t, data)
	got, err := f.GetCellValue("Nils", "A2")
	require.NoError(t, err)
	assert.Equal(t, "", got)
	got, err = f.GetCellValue("Nils", "B2")
	require.NoError(t, err)
	assert.Equal(t, "set", got)
}
