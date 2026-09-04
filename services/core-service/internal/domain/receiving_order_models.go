package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
	"github.com/shopspring/decimal"
)

// ReceivingOrderSummary represents a receiving order in list views.
type ReceivingOrderSummary struct {
	ID                  string
	Number              string
	PurchaseOrderID     string
	PurchaseOrderNumber string
	PurchaseOrderStatus string
	SupplierID          *string
	SupplierName        *string
	SupplierNumber      *string
	LineCount           int32
	Totals              *ReceivingOrderTotals
	Deliveries          []DocumentRef
	CompletedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	// Lines (populated only when the list request includes "lines").
	Lines []*ReceivingOrderLine
}

// ReceivingOrder represents a full receiving order with its lines.
type ReceivingOrder struct {
	ID                  string
	Number              string `audit:"number"`
	PurchaseOrderID     string
	PurchaseOrderNumber string `audit:"purchase_order_number"`
	PurchaseOrderStatus string
	SupplierID          *string `audit:"supplier_id"`
	SupplierName        *string `audit:"supplier_name"`
	SupplierNumber      *string `audit:"supplier_number"`
	Note                *string `audit:"note"`
	Lines               []*ReceivingOrderLine
	Totals              *ReceivingOrderTotals
	Deliveries          []DocumentRef
	CompletedAt         *time.Time `audit:"completed_at"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ReceivingOrderLine represents a line item in a receiving order.
type ReceivingOrderLine struct {
	ID                       string
	QuantityID               string
	QuantityValue            string `audit:"quantity_value"`
	QuantityUnitID           string
	QuantityUnitAbbreviation string  `audit:"quantity_unit_abbreviation"`
	RejectedQuantityValue    *string `audit:"rejected_quantity_value"`
	OrderLineID              string
	OrderLineProductID       *string
	OrderLineItemNumber      *int32
	OrderLineItemID          *string `audit:"order_line_item_id"`
	OrderLineItemSKU         *string `audit:"order_line_item_sku"`
	OrderLineItemDescription *string `audit:"order_line_item_description"`
	// The purchase order line's own quantity record, so the receiving line reports
	// the ordered figure as that quantity rather than as a copy of its value.
	OrderLineQuantityID       string `audit:"order_line_quantity_id"`
	OrderLineQuantityOrdered  string `audit:"order_line_quantity_ordered"`
	OrderLineUnitID           string
	OrderLineUnitAbbreviation string     `audit:"order_line_unit_abbreviation"`
	StockedAt                 *time.Time `audit:"stocked_at"`
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// ReceivingOrderTotals is what a receiving order is worth and how far it has been put away, aggregated over its lines.
//
// Every figure is an amount — the purchase order's agreed unit price times a quantity — because a receiving order's lines can each count in a different unit, and money is the only common denominator they have. Completion is a ratio of two of these amounts for the same reason: summing quantities across lines would add pairs to metres.
type ReceivingOrderTotals struct {
	OrderedAmount  string
	StockedAmount  string
	RejectedAmount string
}

// DocumentRef names one purchasing document from another: the id and number a caller needs to follow the link, plus the status that makes the reference readable without fetching it.
type DocumentRef struct {
	ID     string
	Number string
	Status string
}

// ListReceivingOrdersParams holds parameters for listing receiving orders.
type ListReceivingOrdersParams struct {
	AccountID   string
	Cursor      *string
	Limit       int32
	Query       *string
	Status      *string
	ItemIDs     []string
	SupplierIDs []string
	StartDate   *time.Time
	EndDate     *time.Time
	Includes    []string
}

// ListReceivingOrdersResult holds the result of listing receiving orders.
type ListReceivingOrdersResult struct {
	ReceivingOrders []*ReceivingOrderSummary
	PageInfo        pagination.PageInfo
}

// GetReceivingOrderParams holds parameters for getting a single receiving order.
type GetReceivingOrderParams struct {
	AccountID        string
	ReceivingOrderID string
}

// StockReceivingOrderParams holds parameters for stocking a receiving order.
type StockReceivingOrderParams struct {
	AccountID        string
	ReceivingOrderID string
	Data             StockingData
}

// StockingData contains the stocking data sent in the request body.
type StockingData struct {
	LineItems []StockingLineItem
}

// StockingLineItem represents a single line item in a stocking request.
type StockingLineItem struct {
	ReceivingOrderLineID string
	LotNumber            *string
	RejectedQuantity     *decimal.Decimal
	Allocations          []StorageAllocation
}

// StorageAllocation represents a storage allocation for a stocking line item.
type StorageAllocation struct {
	LocationID *string
	Quantity   decimal.Decimal
}

// UpdateReceivingOrderLineParams holds parameters for updating a receiving order line.
type UpdateReceivingOrderLineParams struct {
	AccountID        string
	ReceivingOrderID string
	LineID           string
	QuantityValue    *string
}

// UnstockedLine holds the ID and order line ID of an unstocked line.
type UnstockedLine struct {
	ID          string
	OrderLineID string
}

// ReceivingOrderLineUnitPrice holds unit price information for a receiving order line.
type ReceivingOrderLineUnitPrice struct {
	ReceivingOrderLineID       string
	ItemID                     string
	UnitPriceValue             string
	UnitPriceNumeratorUnitID   string
	UnitPriceDenominatorUnitID string
	QuantityUnitID             string
}

// OpenInventoryIssue represents an open inventory issue for FIFO allocation.
type OpenInventoryIssue struct {
	ID            string
	QuantityID    string
	QuantityValue string
	UnitID        string
	LocationID    *string
	LotID         *string
}
