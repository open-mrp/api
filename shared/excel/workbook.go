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

// describes a worksheet whose rows are produced while it is written, so the caller never holds them all
type StreamSheet struct {
	Name    string
	Columns []ColumnSpec
	// Fill produces the rows in order, handing each to write.
	Fill func(write func(Row) error) error
}

// is how many data rows one worksheet takes under its header. A variable so a test can fill a sheet without writing a million rows.
var sheetRowLimit = excelize.TotalRows - 1

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

	boldID, err := headerStyle(f)
	if err != nil {
		return nil, err
	}

	for _, sheet := range spec.Sheets {
		if err := writeSheet(f, sheet.Name, sheet.Columns, boldID, sheet.fill); err != nil {
			return nil, err
		}
	}

	return serialize(f)
}

// renders a sheet whose rows arrive while it is written. Rows past what one worksheet holds continue on further sheets, numbered after the first ("Logs 2").
func Stream(sheet StreamSheet) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := f.SetSheetName("Sheet1", sheet.Name); err != nil {
		return nil, fmt.Errorf("name sheet %q: %w", sheet.Name, err)
	}
	boldID, err := headerStyle(f)
	if err != nil {
		return nil, err
	}
	if err := writeSheet(f, sheet.Name, sheet.Columns, boldID, sheet.Fill); err != nil {
		return nil, err
	}

	return serialize(f)
}

func headerStyle(f *excelize.File) (int, error) {
	id, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return 0, fmt.Errorf("create header style: %w", err)
	}
	return id, nil
}

func serialize(f *excelize.File) ([]byte, error) {
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("serialize workbook: %w", err)
	}
	return buf.Bytes(), nil
}

// hands a held sheet's rows to write, in order
func (s Sheet) fill(write func(Row) error) error {
	for _, row := range s.Rows {
		if err := write(row); err != nil {
			return err
		}
	}
	return nil
}

// writes a sheet's rows as fill produces them, continuing on a new sheet whenever one fills
func writeSheet(f *excelize.File, name string, columns []ColumnSpec, boldID int, fill func(write func(Row) error) error) error {
	index, err := columnIndex(name, columns)
	if err != nil {
		return err
	}
	numFmtIDs, err := columnStyles(f, columns)
	if err != nil {
		return err
	}

	current := name
	sw, err := startSheet(f, current, columns, index, boldID)
	if err != nil {
		return err
	}
	part, written := 1, 0
	err = fill(func(row Row) error {
		if written == sheetRowLimit {
			if err := sw.Flush(); err != nil {
				return fmt.Errorf("flush sheet %q: %w", current, err)
			}
			part++
			current = continuationName(name, part)
			if _, err := f.NewSheet(current); err != nil {
				return fmt.Errorf("create sheet %q: %w", current, err)
			}
			if sw, err = startSheet(f, current, columns, index, boldID); err != nil {
				return err
			}
			written = 0
		}

		cells := make([]any, len(columns))
		for i, c := range columns {
			cells[i] = excelize.Cell{StyleID: numFmtIDs[c.Key], Value: row[c.Key]}
		}
		anchor, err := cellName(0, written+2)
		if err != nil {
			return err
		}
		if err := sw.SetRow(anchor, cells); err != nil {
			return fmt.Errorf("write row %d of %q: %w", written+2, current, err)
		}
		written++
		return nil
	})
	if err != nil {
		return err
	}

	if err := sw.Flush(); err != nil {
		return fmt.Errorf("flush sheet %q: %w", current, err)
	}
	return nil
}

// opens a sheet for streaming in the only order excelize permits: widths and comments
// before Flush seals the sheet, and widths before the first row
func startSheet(f *excelize.File, name string, columns []ColumnSpec, index map[string]int, boldID int) (*excelize.StreamWriter, error) {
	sw, err := f.NewStreamWriter(name)
	if err != nil {
		return nil, fmt.Errorf("stream sheet %q: %w", name, err)
	}

	for i, c := range columns {
		if c.Width <= 0 {
			continue
		}
		if err := sw.SetColWidth(i+1, i+1, c.Width); err != nil {
			return nil, fmt.Errorf("set width for %q: %w", c.Key, err)
		}
	}

	if err := applyNotes(f, name, columns, index); err != nil {
		return nil, err
	}

	header := make([]any, len(columns))
	for i, c := range columns {
		header[i] = excelize.Cell{StyleID: boldID, Value: c.Header}
	}
	if err := sw.SetRow("A1", header); err != nil {
		return nil, fmt.Errorf("write header for %q: %w", name, err)
	}
	return sw, nil
}

// names the sheet a stream continues on, trimming the base so the name stays within Excel's limit
func continuationName(name string, part int) string {
	suffix := fmt.Sprintf(" %d", part)
	if base := []rune(name); len(base)+len(suffix) > excelize.MaxSheetNameLength {
		name = string(base[:excelize.MaxSheetNameLength-len(suffix)])
	}
	return name + suffix
}

// maps each column key to its position, rejecting the mistakes that would
// silently misplace a cell
func columnIndex(name string, columns []ColumnSpec) (map[string]int, error) {
	if len(columns) == 0 {
		return nil, fmt.Errorf("excel: sheet %q has no columns", name)
	}
	index := make(map[string]int, len(columns))
	for i, c := range columns {
		if c.Key == "" {
			return nil, fmt.Errorf("excel: sheet %q column %d has no key", name, i)
		}
		if _, dup := index[c.Key]; dup {
			return nil, fmt.Errorf("excel: sheet %q has duplicate column key %q", name, c.Key)
		}
		index[c.Key] = i
	}
	return index, nil
}

// comments the header cell of every column that declares a note
func applyNotes(f *excelize.File, name string, columns []ColumnSpec, index map[string]int) error {
	for _, c := range columns {
		if c.Note == "" {
			continue
		}
		cell, err := cellName(index[c.Key], 1)
		if err != nil {
			return err
		}
		if err := f.AddComment(name, excelize.Comment{Cell: cell, Text: c.Note}); err != nil {
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
