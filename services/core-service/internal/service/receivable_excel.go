package service

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/augno/api/services/core-service/internal/domain"
)

const (
	statementSheet       = "Statement of Account"
	currencyNumberFormat = `_($* #,##0.00_);_($* (#,##0.00);_($* "-"??_);_(@_)`
)

// GenerateStatementOfAccount generates an Excel workbook containing a statement of account with aging buckets for the given receivable invoices and open credits.
func GenerateStatementOfAccount(receivables []domain.ReceivableEntry, openCredits []domain.OpenCredit) ([]byte, error) {
	f := excelize.NewFile()

	idx, err := f.NewSheet(statementSheet)
	if err != nil {
		return nil, fmt.Errorf("create sheet: %w", err)
	}
	_ = f.DeleteSheet("Sheet1")
	f.SetActiveSheet(idx)

	// Define headers and column widths.
	type column struct {
		header string
		width  float64
	}
	columns := []column{
		{"Invoice Number", 20},
		{"Purchase Order Number", 20},
		{"Invoice Date", 15},
		{"Current", 15},
		{"Over 30 Days", 15},
		{"Over 60 Days", 15},
		{"Over 90 Days", 15},
		{"Over 120 Days", 15},
		{"Total", 15},
	}

	// Write headers and set column widths.
	for i, col := range columns {
		cellName, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(statementSheet, cellName, col.header); err != nil {
			return nil, fmt.Errorf("set header %q: %w", col.header, err)
		}
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		if err := f.SetColWidth(statementSheet, colLetter, colLetter, col.width); err != nil {
			return nil, fmt.Errorf("set column width: %w", err)
		}
	}

	// Create currency style for columns D-I (columns 4-9).
	currencyStyle, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: new(currencyNumberFormat),
	})
	if err != nil {
		return nil, fmt.Errorf("create currency style: %w", err)
	}

	// Sort receivables by invoiced date descending.
	sorted := make([]domain.ReceivableEntry, len(receivables))
	copy(sorted, receivables)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].InvoicedAt.After(sorted[j].InvoicedAt)
	})

	now := time.Now()

	// Track totals for the summary row.
	var totalCurrent, totalOver30, totalOver60, totalOver90, totalOver120, totalAll float64

	row := 2 // Row 1 is headers.

	// Write receivable invoice rows.
	for _, entry := range sorted {
		daysDiff := int(now.Sub(entry.InvoicedAt).Hours() / 24)

		balance, err := strconv.ParseFloat(entry.RemainingBalance, 64)
		if err != nil {
			return nil, fmt.Errorf("parse remaining balance %q: %w", entry.RemainingBalance, err)
		}

		poNumber := ""
		if entry.PONumber != nil {
			poNumber = *entry.PONumber
		}

		current, over30, over60, over90, over120 := bucketAmount(daysDiff, balance)

		totalCurrent += current
		totalOver30 += over30
		totalOver60 += over60
		totalOver90 += over90
		totalOver120 += over120
		totalAll += balance

		if err := writeStatementRow(f, row, entry.InvoiceNumber, poNumber, entry.InvoicedAt, current, over30, over60, over90, over120, balance); err != nil {
			return nil, fmt.Errorf("write invoice row: %w", err)
		}
		row++
	}

	// Write open credit rows.
	for _, credit := range openCredits {
		daysDiff := int(now.Sub(credit.CreatedAt).Hours() / 24)

		leftover, err := strconv.ParseFloat(credit.LeftoverAmount, 64)
		if err != nil {
			return nil, fmt.Errorf("parse leftover amount %q: %w", credit.LeftoverAmount, err)
		}

		negated := -leftover
		current, over30, over60, over90, over120 := bucketAmount(daysDiff, negated)

		totalCurrent += current
		totalOver30 += over30
		totalOver60 += over60
		totalOver90 += over90
		totalOver120 += over120
		totalAll += negated

		numberLabel := credit.Number
		if numberLabel == "" {
			numberLabel = "N/A"
		}
		invoiceLabel := fmt.Sprintf("Credit: %s", numberLabel)

		if err := writeStatementRow(f, row, invoiceLabel, "", credit.CreatedAt, current, over30, over60, over90, over120, negated); err != nil {
			return nil, fmt.Errorf("write credit row: %w", err)
		}
		row++
	}

	// Write totals row.
	if err := writeStatementRow(f, row, "", "", time.Time{}, totalCurrent, totalOver30, totalOver60, totalOver90, totalOver120, totalAll); err != nil {
		return nil, fmt.Errorf("write totals row: %w", err)
	}
	// Set "Totals" label in the Invoice Date column (column 3).
	totalsDateCell, _ := excelize.CoordinatesToCellName(3, row)
	if err := f.SetCellValue(statementSheet, totalsDateCell, "Totals"); err != nil {
		return nil, fmt.Errorf("set totals label: %w", err)
	}

	// Style the totals row: bold + light gray fill.
	totalsStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"DDDDDD"},
		},
		CustomNumFmt: new(currencyNumberFormat),
	})
	if err != nil {
		return nil, fmt.Errorf("create totals style: %w", err)
	}
	totalsTextStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"DDDDDD"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create totals text style: %w", err)
	}

	// Apply bold+fill to text columns (A-C).
	for col := 1; col <= 3; col++ {
		cellName, _ := excelize.CoordinatesToCellName(col, row)
		if err := f.SetCellStyle(statementSheet, cellName, cellName, totalsTextStyle); err != nil {
			return nil, fmt.Errorf("set totals text style: %w", err)
		}
	}
	// Apply bold+fill+currency to numeric columns (D-I).
	for col := 4; col <= 9; col++ {
		cellName, _ := excelize.CoordinatesToCellName(col, row)
		if err := f.SetCellStyle(statementSheet, cellName, cellName, totalsStyle); err != nil {
			return nil, fmt.Errorf("set totals numeric style: %w", err)
		}
	}

	// Apply currency format to all data rows (columns D-I, rows 2 through row-1).
	if row > 2 {
		for col := 4; col <= 9; col++ {
			for r := 2; r < row; r++ {
				cellName, _ := excelize.CoordinatesToCellName(col, r)
				if err := f.SetCellStyle(statementSheet, cellName, cellName, currencyStyle); err != nil {
					return nil, fmt.Errorf("set currency style: %w", err)
				}
			}
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
	}
	return buf.Bytes(), nil
}

// bucketAmount assigns the given amount to the appropriate aging bucket based on daysDiff.
func bucketAmount(daysDiff int, amount float64) (current, over30, over60, over90, over120 float64) {
	switch {
	case daysDiff <= 30:
		current = amount
	case daysDiff <= 60:
		over30 = amount
	case daysDiff <= 90:
		over60 = amount
	case daysDiff <= 120:
		over90 = amount
	default:
		over120 = amount
	}
	return
}

// writeStatementRow writes a single row of data to the statement sheet.
func writeStatementRow(f *excelize.File, row int, invoiceNumber, poNumber string, date time.Time, current, over30, over60, over90, over120, total float64) error {
	setCellStr := func(col int, val string) error {
		cellName, _ := excelize.CoordinatesToCellName(col, row)
		return f.SetCellValue(statementSheet, cellName, val)
	}
	setCellFloat := func(col int, val float64) error {
		cellName, _ := excelize.CoordinatesToCellName(col, row)
		return f.SetCellValue(statementSheet, cellName, val)
	}

	if err := setCellStr(1, invoiceNumber); err != nil {
		return err
	}
	if err := setCellStr(2, poNumber); err != nil {
		return err
	}
	// Column 3: Invoice Date — only set if date is non-zero.
	if !date.IsZero() {
		cellName, _ := excelize.CoordinatesToCellName(3, row)
		if err := f.SetCellValue(statementSheet, cellName, date.Format("1/2/2006")); err != nil {
			return err
		}
	}
	if err := setCellFloat(4, current); err != nil {
		return err
	}
	if err := setCellFloat(5, over30); err != nil {
		return err
	}
	if err := setCellFloat(6, over60); err != nil {
		return err
	}
	if err := setCellFloat(7, over90); err != nil {
		return err
	}
	if err := setCellFloat(8, over120); err != nil {
		return err
	}
	if err := setCellFloat(9, total); err != nil {
		return err
	}
	return nil
}
