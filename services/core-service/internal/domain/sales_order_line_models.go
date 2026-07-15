package domain

import (
	"time"
)

// SalesOrderLine represents a sales order line domain model.
type SalesOrderLine struct {
	ID                 string
	LineItemNumber     int32   `audit:"line_item_number"`
	ProductSKU         string  `audit:"product_sku"`
	ProductDescription *string `audit:"product_description"`
	ProductID          *string `audit:"product_id"`
	ProductTypeCode    *string `audit:"product_type_code"`
	ItemID             *string `audit:"item_id"`
	ItemSKU            *string `audit:"item_sku"`
	SalesOrderID       string
	EdiLineItemID      *string `audit:"edi_line_item_id"`

	// Quantity ordered
	QuantityID               string
	QuantityValue            string `audit:"quantity_value"`
	QuantityUnitID           string `audit:"quantity_unit_id"`
	QuantityUnitName         string `audit:"quantity_unit_name"`
	QuantityUnitAbbreviation string `audit:"quantity_unit_abbreviation"`
	QuantityUnitType         string `audit:"quantity_unit_type"`

	// Aggregated quantity values
	QuantityPickedValue   *string `audit:"quantity_picked_value"`
	QuantityPackedValue   *string `audit:"quantity_packed_value"`
	QuantityInvoicedValue *string `audit:"quantity_invoiced_value"`

	// Unit price
	UnitPriceID                  string
	UnitPriceValue               string `audit:"unit_price_value"`
	UnitPriceNumeratorUnitID     string `audit:"unit_price_numerator_unit_id"`
	UnitPriceNumeratorUnitAbbr   string `audit:"unit_price_numerator_unit_abbr"`
	UnitPriceDenominatorUnitID   string `audit:"unit_price_denominator_unit_id"`
	UnitPriceDenominatorUnitAbbr string `audit:"unit_price_denominator_unit_abbr"`

	// Unit cost (nullable)
	UnitCostID                  *string
	UnitCostValue               *string `audit:"unit_cost_value"`
	UnitCostNumeratorUnitID     *string `audit:"unit_cost_numerator_unit_id"`
	UnitCostNumeratorUnitAbbr   *string `audit:"unit_cost_numerator_unit_abbr"`
	UnitCostDenominatorUnitID   *string `audit:"unit_cost_denominator_unit_id"`
	UnitCostDenominatorUnitAbbr *string `audit:"unit_cost_denominator_unit_abbr"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateSalesOrderLineParams holds the parameters for creating a sales order line.
type CreateSalesOrderLineParams struct {
	SalesOrderID               string
	AccountID                  string
	ProductID                  string
	ItemID                     *string
	ProductSKU                 string
	ProductDescription         *string
	QuantityValue              string
	QuantityUnitID             string
	UnitPriceValue             string
	UnitPriceNumeratorUnitID   string
	UnitPriceDenominatorUnitID string
	UnitCostValue              *string
	UnitCostNumeratorUnitID    *string
	UnitCostDenominatorUnitID  *string
	EdiLineItemID              *string
}

// UpdateSalesOrderLineParams holds the parameters for updating a sales order line.
type UpdateSalesOrderLineParams struct {
	SalesOrderLineID           string
	SalesOrderID               string
	AccountID                  string
	ProductID                  *string
	ItemID                     *string
	ProductSKU                 *string
	ProductDescription         *string
	QuantityValue              *string
	QuantityUnitID             *string
	UnitPriceValue             *string
	UnitPriceNumeratorUnitID   *string
	UnitPriceDenominatorUnitID *string
	UnitCostValue              *string
	UnitCostNumeratorUnitID    *string
	UnitCostDenominatorUnitID  *string
	EdiLineItemID              *string
}

// DeleteSalesOrderLineParams holds the parameters for deleting a sales order line.
type DeleteSalesOrderLineParams struct {
	SalesOrderLineID string
	SalesOrderID     string
	AccountID        string
}

// ReorderSalesOrderLinesParams holds the parameters for re-sequencing a sales order's lines.
type ReorderSalesOrderLinesParams struct {
	SalesOrderID string
	AccountID    string
	// LineIDs are the order's product-line IDs in the desired display order. Credit/freight lines are kept at the bottom and must not appear here.
	LineIDs []string
}

// SalesOrderLinePosition is a line's current position and whether it is a credit/freight (system) line, used when re-sequencing.
type SalesOrderLinePosition struct {
	ID             string
	LineItemNumber int32
	IsSystem       bool
}
