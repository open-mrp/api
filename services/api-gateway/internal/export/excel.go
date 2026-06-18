package export

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	pb "github.com/augno/api/shared/proto/core"
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

type propMeta struct {
	id   string
	name string
}

// collectProperties collects unique properties (in encounter order) from a slice of items' category property lists.
func collectProperties(items []*apiresource.Item) ([]string, map[string]propMeta) {
	order := []string{}
	byID := map[string]propMeta{}
	for _, item := range items {
		if item == nil || item.Category == nil || item.Category.Properties == nil {
			continue
		}
		for _, prop := range item.Category.Properties.Data {
			if _, exists := byID[prop.ID]; !exists {
				byID[prop.ID] = propMeta{id: prop.ID, name: prop.Name}
				order = append(order, prop.ID)
			}
		}
	}
	return order, byID
}

// attributeByPropertyID builds a map from property ID → attribute value for a single item, using the item's attributes list.
func attributeByPropertyID(item *apiresource.Item) map[string]string {
	result := map[string]string{}
	if item == nil || item.Attributes == nil {
		return result
	}
	for _, attr := range item.Attributes.Data {
		if attr.Property != nil {
			result[attr.Property.ID] = attr.Value
		}
	}
	return result
}

func itemFields(item *apiresource.Item) (sku, description, categoryName, unitPriceValue, unitPriceUnit, unitCostValue, unitCostUnit string) {
	if item == nil {
		return
	}
	sku = item.SKU
	if item.Description != nil {
		description = *item.Description
	}
	if item.Category != nil {
		categoryName = item.Category.Name
	}
	if item.UnitValue != nil {
		unitPriceValue = item.UnitValue.Value
		if item.UnitValue.NumeratorUnit != nil && item.UnitValue.DenominatorUnit != nil {
			unitPriceUnit = item.UnitValue.NumeratorUnit.Abbreviation + "/" + item.UnitValue.DenominatorUnit.Abbreviation
		}
	}
	if item.UnitCost != nil {
		unitCostValue = item.UnitCost.Value
		if item.UnitCost.DenominatorUnit != nil {
			unitCostUnit = item.UnitCost.DenominatorUnit.Abbreviation
		}
	}
	return
}

// ProductsToExcel builds an Excel file from a slice of products.
func ProductsToExcel(products []apiresource.Product) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Products"
	if len(products) == 0 {
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

	items := make([]*apiresource.Item, len(products))
	for i, p := range products {
		items[i] = p.Item
	}
	propOrder, propByID := collectProperties(items)

	baseHeaders := []string{"ID", "SKU", "Description", "Category", "Product Line", "Unit Price", "Price Unit", "Unit Cost", "Cost Unit"}
	headers := make([]string, len(baseHeaders)+len(propOrder))
	copy(headers, baseHeaders)
	for i, pID := range propOrder {
		headers[len(baseHeaders)+i] = propByID[pID].name
	}
	for i, h := range headers {
		c, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, c, h)
	}
	lastCol, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetCellStyle(sheet, "A1", lastCol+"1", boldStyle(f))

	for row, p := range products {
		r := row + 2
		sku, description, categoryName, unitPriceValue, unitPriceUnit, unitCostValue, unitCostUnit := itemFields(p.Item)
		productLineName := ""
		if p.ProductLine != nil {
			productLineName = p.ProductLine.Name
		}
		_ = f.SetCellValue(sheet, cell(r, 1), p.ID)
		_ = f.SetCellValue(sheet, cell(r, 2), sku)
		_ = f.SetCellValue(sheet, cell(r, 3), description)
		_ = f.SetCellValue(sheet, cell(r, 4), categoryName)
		_ = f.SetCellValue(sheet, cell(r, 5), productLineName)
		_ = f.SetCellValue(sheet, cell(r, 6), unitPriceValue)
		_ = f.SetCellValue(sheet, cell(r, 7), unitPriceUnit)
		_ = f.SetCellValue(sheet, cell(r, 8), unitCostValue)
		_ = f.SetCellValue(sheet, cell(r, 9), unitCostUnit)
		attrByProp := attributeByPropertyID(p.Item)
		for i, pID := range propOrder {
			_ = f.SetCellValue(sheet, cell(r, len(baseHeaders)+i+1), attrByProp[pID])
		}
	}
	return writeExcel(f)
}

// MaterialsToExcel builds an Excel file from a slice of materials.
func MaterialsToExcel(materials []apiresource.Material) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Materials"
	if len(materials) == 0 {
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

	items := make([]*apiresource.Item, len(materials))
	for i, m := range materials {
		items[i] = m.Item
	}
	propOrder, propByID := collectProperties(items)

	baseHeaders := []string{"ID", "SKU", "Description", "Category", "Unit Price", "Price Unit", "Unit Cost", "Cost Unit"}
	headers := make([]string, len(baseHeaders)+len(propOrder))
	copy(headers, baseHeaders)
	for i, pID := range propOrder {
		headers[len(baseHeaders)+i] = propByID[pID].name
	}
	for i, h := range headers {
		c, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, c, h)
	}
	lastCol, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetCellStyle(sheet, "A1", lastCol+"1", boldStyle(f))

	for row, m := range materials {
		r := row + 2
		sku, description, categoryName, unitPriceValue, unitPriceUnit, unitCostValue, unitCostUnit := itemFields(m.Item)
		_ = f.SetCellValue(sheet, cell(r, 1), m.ID)
		_ = f.SetCellValue(sheet, cell(r, 2), sku)
		_ = f.SetCellValue(sheet, cell(r, 3), description)
		_ = f.SetCellValue(sheet, cell(r, 4), categoryName)
		_ = f.SetCellValue(sheet, cell(r, 5), unitPriceValue)
		_ = f.SetCellValue(sheet, cell(r, 6), unitPriceUnit)
		_ = f.SetCellValue(sheet, cell(r, 7), unitCostValue)
		_ = f.SetCellValue(sheet, cell(r, 8), unitCostUnit)
		attrByProp := attributeByPropertyID(m.Item)
		for i, pID := range propOrder {
			_ = f.SetCellValue(sheet, cell(r, len(baseHeaders)+i+1), attrByProp[pID])
		}
	}
	return writeExcel(f)
}

// PartsToExcel builds an Excel file from a slice of parts.
func PartsToExcel(parts []apiresource.Part) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Parts"
	if len(parts) == 0 {
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

	items := make([]*apiresource.Item, len(parts))
	for i, p := range parts {
		items[i] = p.Item
	}
	propOrder, propByID := collectProperties(items)

	baseHeaders := []string{"ID", "SKU", "Description", "Category", "Unit Price", "Price Unit", "Unit Cost", "Cost Unit"}
	headers := make([]string, len(baseHeaders)+len(propOrder))
	copy(headers, baseHeaders)
	for i, pID := range propOrder {
		headers[len(baseHeaders)+i] = propByID[pID].name
	}
	for i, h := range headers {
		c, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, c, h)
	}
	lastCol, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetCellStyle(sheet, "A1", lastCol+"1", boldStyle(f))

	for row, p := range parts {
		r := row + 2
		sku, description, categoryName, unitPriceValue, unitPriceUnit, unitCostValue, unitCostUnit := itemFields(p.Item)
		_ = f.SetCellValue(sheet, cell(r, 1), p.ID)
		_ = f.SetCellValue(sheet, cell(r, 2), sku)
		_ = f.SetCellValue(sheet, cell(r, 3), description)
		_ = f.SetCellValue(sheet, cell(r, 4), categoryName)
		_ = f.SetCellValue(sheet, cell(r, 5), unitPriceValue)
		_ = f.SetCellValue(sheet, cell(r, 6), unitPriceUnit)
		_ = f.SetCellValue(sheet, cell(r, 7), unitCostValue)
		_ = f.SetCellValue(sheet, cell(r, 8), unitCostUnit)
		attrByProp := attributeByPropertyID(p.Item)
		for i, pID := range propOrder {
			_ = f.SetCellValue(sheet, cell(r, len(baseHeaders)+i+1), attrByProp[pID])
		}
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
