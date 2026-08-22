package export

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/open-mrp/api/shared/proto/core"
)

const (
	ExcelContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
)

// ItemsToExcel builds an Excel file from an export items response.
func ItemsToExcel(resp *pb.ExportItemsResponse) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Items"
	if resp == nil || len(resp.Items) == 0 {
		if err := f.SetSheetName("Sheet1", sheet); err != nil {
			return nil, err
		}
		return writeExcel(f)
	}

	idx, err := f.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	_ = f.DeleteSheet("Sheet1")
	f.SetActiveSheet(idx)

	headers := []string{"ID", "SKU", "Description", "Notes", "Item Type", "Category", "On Hand Qty", "Unit", "Created At", "Updated At"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	_ = f.SetCellStyle(sheet, "A1", "J1", boldStyle(f))

	for row, item := range resp.Items {
		r := row + 2
		_ = f.SetCellValue(sheet, cell(r, 1), item.Id)
		_ = f.SetCellValue(sheet, cell(r, 2), item.Sku)
		_ = f.SetCellValue(sheet, cell(r, 3), ptrStr(item.Description))
		_ = f.SetCellValue(sheet, cell(r, 4), ptrStr(item.Notes))
		_ = f.SetCellValue(sheet, cell(r, 5), item.ItemTypeCode)
		_ = f.SetCellValue(sheet, cell(r, 6), item.CategoryName)
		_ = f.SetCellValue(sheet, cell(r, 7), item.OnHandQuantity)
		_ = f.SetCellValue(sheet, cell(r, 8), item.OnHandUnitId)
		_ = f.SetCellValue(sheet, cell(r, 9), formatTime(item.CreatedAt))
		_ = f.SetCellValue(sheet, cell(r, 10), formatTime(item.UpdatedAt))
	}

	return writeExcel(f)
}

// InventoryChangeLogsToExcel builds an Excel file from an export inventory change logs response.
func InventoryChangeLogsToExcel(resp *pb.ExportInventoryChangeLogsResponse) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Inventory Change Logs"
	if resp == nil || len(resp.InventoryChangeLogs) == 0 {
		if err := f.SetSheetName("Sheet1", sheet); err != nil {
			return nil, err
		}
		return writeExcel(f)
	}

	idx, err := f.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	_ = f.DeleteSheet("Sheet1")
	f.SetActiveSheet(idx)

	headers := []string{"Item", "Quantity Change", "Unit", "Action Type", "Responsible User", "Responsible Scanning Station", "Created At"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	_ = f.SetCellStyle(sheet, "A1", "G1", boldStyle(f))

	for row, log := range resp.InventoryChangeLogs {
		r := row + 2
		_ = f.SetCellValue(sheet, cell(r, 1), log.ItemSku)
		_ = f.SetCellValue(sheet, cell(r, 2), log.QuantityValue)
		_ = f.SetCellValue(sheet, cell(r, 3), log.QuantityUnitAbbreviation)
		_ = f.SetCellValue(sheet, cell(r, 4), log.ActionTypeCode)
		_ = f.SetCellValue(sheet, cell(r, 5), ptrStr(log.ResponsibleUserName))
		_ = f.SetCellValue(sheet, cell(r, 6), ptrStr(log.ScanningStationName))
		_ = f.SetCellValue(sheet, cell(r, 7), formatTime(log.CreatedAt))
	}

	return writeExcel(f)
}

func cell(row, col int) string {
	c, _ := excelize.CoordinatesToCellName(col, row)
	return c
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatTime(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	t := ts.AsTime()
	return t.Format(time.RFC3339)
}

func boldStyle(f *excelize.File) int {
	style, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	return style
}

func writeExcel(f *excelize.File) ([]byte, error) {
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("excel write: %w", err)
	}
	return buf.Bytes(), nil
}
