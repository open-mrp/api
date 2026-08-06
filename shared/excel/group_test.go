package excel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestGroup(t *testing.T) {
	parent := Row{"id": "run_1", "number": "R-1"}

	tests := []struct {
		name     string
		children []Row
		want     []Row
	}{
		{
			name:     "a childless parent still emits one row",
			children: nil,
			want:     []Row{{"id": "run_1", "number": "R-1"}},
		},
		{
			name:     "a single child merges onto the parent's row",
			children: []Row{{"batch": "B-1"}},
			want:     []Row{{"id": "run_1", "number": "R-1", "batch": "B-1"}},
		},
		{
			name:     "later children omit the parent's keys",
			children: []Row{{"batch": "B-1"}, {"batch": "B-2"}, {"batch": "B-3"}},
			want: []Row{
				{"id": "run_1", "number": "R-1", "batch": "B-1"},
				{"batch": "B-2"},
				{"batch": "B-3"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Group(parent, tc.children))
		})
	}
}

// the parent must survive being grouped more than once
func TestGroup_DoesNotMutateTheParent(t *testing.T) {
	parent := Row{"id": "run_1"}

	first := Group(parent, []Row{{"batch": "B-1"}})
	second := Group(parent, []Row{{"batch": "B-2"}})

	assert.Equal(t, Row{"id": "run_1"}, parent, "the caller's row is untouched")
	assert.Equal(t, "B-1", first[0]["batch"])
	assert.Equal(t, "B-2", second[0]["batch"])
}

// what the blanking actually has to look like once written to a sheet
func TestGroup_ContinuationRowsRenderBlank(t *testing.T) {
	sheet := Sheet{
		Name: "Production Runs",
		Columns: []ColumnSpec{
			{Header: "ID", Key: "id", Width: 10},
			{Header: "Number", Key: "number", Width: 10},
			{Header: "Batch", Key: "batch", Width: 10},
		},
	}
	sheet.Rows = append(sheet.Rows, Group(Row{"id": "run_1", "number": "R-1"},
		[]Row{{"batch": "B-1"}, {"batch": "B-2"}})...)
	sheet.Rows = append(sheet.Rows, Group(Row{"id": "run_2", "number": "R-2"}, nil)...)

	data, err := Build(Spec{Sheets: []Sheet{sheet}})
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	for cell, want := range map[string]string{
		"A2": "run_1", "B2": "R-1", "C2": "B-1",
		"A3": "", "B3": "", "C3": "B-2", // continuation row
		"A4": "run_2", "B4": "R-2", "C4": "", // childless parent
	} {
		got, err := f.GetCellValue("Production Runs", cell)
		require.NoError(t, err)
		assert.Equal(t, want, got, "cell %s", cell)
	}
}
