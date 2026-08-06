// Package excel builds spreadsheet workbooks from a declarative description of
// columns and rows, with no knowledge of any domain type.
package excel

import (
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// identifies the workbooks this package produces
const ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// describes one column. Rows address it by Key; nothing outside this package
// computes a column letter, which is what stops the two diverging.
type ColumnSpec struct {
	Header string
	Key    string
	Width  float64
	// NumFmt is an Excel custom number format applied to the column's data cells.
	NumFmt string
	// Note is a hover comment on the header cell.
	Note string
}

// carries one row's cells keyed by ColumnSpec.Key; a missing key is blank
type Row map[string]any

// describes one worksheet
type Sheet struct {
	Name    string
	Columns []ColumnSpec
	Rows    []Row
}

// describes a whole workbook
type Spec struct {
	Sheets []Sheet
}

// reports a spec that would produce a file Excel refuses to open
var ErrNoSheets = errors.New("excel: a workbook needs at least one sheet")

// renders a spec to xlsx bytes
func Build(spec Spec) ([]byte, error) {
	if len(spec.Sheets) == 0 {
		return nil, ErrNoSheets
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	for i, sheet := range spec.Sheets {
		idx, err := f.NewSheet(sheet.Name)
		if err != nil {
			return nil, fmt.Errorf("create sheet %q: %w", sheet.Name, err)
		}
		if i == 0 {
			f.SetActiveSheet(idx)
		}
	}
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return nil, fmt.Errorf("remove default sheet: %w", err)
	}

	boldID, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, fmt.Errorf("create header style: %w", err)
	}

	for _, sheet := range spec.Sheets {
		if err := writeSheet(f, sheet, boldID); err != nil {
			return nil, err
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("serialize workbook: %w", err)
	}
	return buf.Bytes(), nil
}

// writes one sheet in the only order excelize permits: widths and comments before
// Flush seals the sheet, and widths before the first row
func writeSheet(f *excelize.File, sheet Sheet, boldID int) error {
	index, err := columnIndex(sheet)
	if err != nil {
		return err
	}

	sw, err := f.NewStreamWriter(sheet.Name)
	if err != nil {
		return fmt.Errorf("stream sheet %q: %w", sheet.Name, err)
	}

	for i, c := range sheet.Columns {
		if c.Width <= 0 {
			continue
		}
		if err := sw.SetColWidth(i+1, i+1, c.Width); err != nil {
			return fmt.Errorf("set width for %q: %w", c.Key, err)
		}
	}

	if err := applyNotes(f, sheet, index); err != nil {
		return err
	}

	numFmtIDs, err := columnStyles(f, sheet.Columns)
	if err != nil {
		return err
	}

	header := make([]any, len(sheet.Columns))
	for i, c := range sheet.Columns {
		header[i] = excelize.Cell{StyleID: boldID, Value: c.Header}
	}
	if err := sw.SetRow("A1", header); err != nil {
		return fmt.Errorf("write header for %q: %w", sheet.Name, err)
	}

	for r, row := range sheet.Rows {
		cells := make([]any, len(sheet.Columns))
		for i, c := range sheet.Columns {
			cells[i] = excelize.Cell{StyleID: numFmtIDs[c.Key], Value: row[c.Key]}
		}
		anchor, err := cellName(0, r+2)
		if err != nil {
			return err
		}
		if err := sw.SetRow(anchor, cells); err != nil {
			return fmt.Errorf("write row %d of %q: %w", r+2, sheet.Name, err)
		}
	}

	if err := sw.Flush(); err != nil {
		return fmt.Errorf("flush sheet %q: %w", sheet.Name, err)
	}
	return nil
}

// maps each column key to its position, rejecting the mistakes that would
// silently misplace a cell
func columnIndex(sheet Sheet) (map[string]int, error) {
	if len(sheet.Columns) == 0 {
		return nil, fmt.Errorf("excel: sheet %q has no columns", sheet.Name)
	}
	index := make(map[string]int, len(sheet.Columns))
	for i, c := range sheet.Columns {
		if c.Key == "" {
			return nil, fmt.Errorf("excel: sheet %q column %d has no key", sheet.Name, i)
		}
		if _, dup := index[c.Key]; dup {
			return nil, fmt.Errorf("excel: sheet %q has duplicate column key %q", sheet.Name, c.Key)
		}
		index[c.Key] = i
	}
	return index, nil
}

// comments the header cell of every column that declares a note
func applyNotes(f *excelize.File, sheet Sheet, index map[string]int) error {
	for _, c := range sheet.Columns {
		if c.Note == "" {
			continue
		}
		cell, err := cellName(index[c.Key], 1)
		if err != nil {
			return err
		}
		if err := f.AddComment(sheet.Name, excelize.Comment{Cell: cell, Text: c.Note}); err != nil {
			return fmt.Errorf("add note to %q: %w", c.Key, err)
		}
	}
	return nil
}

// builds one style per column that declares a number format
func columnStyles(f *excelize.File, columns []ColumnSpec) (map[string]int, error) {
	ids := make(map[string]int, len(columns))
	for _, c := range columns {
		if c.NumFmt == "" {
			continue
		}
		format := c.NumFmt
		id, err := f.NewStyle(&excelize.Style{CustomNumFmt: &format})
		if err != nil {
			return nil, fmt.Errorf("create number format for %q: %w", c.Key, err)
		}
		ids[c.Key] = id
	}
	return ids, nil
}

// names the cell at a zero-based column and one-based row
func cellName(col, row int) (string, error) {
	name, err := excelize.CoordinatesToCellName(col+1, row)
	if err != nil {
		return "", fmt.Errorf("resolve cell: %w", err)
	}
	return name, nil
}
