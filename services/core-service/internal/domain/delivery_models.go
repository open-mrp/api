package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// DeliverySummary represents a delivery with line count instead of full lines.
type DeliverySummary struct {
	ID                  string
	Number              string
	PurchaseOrderID     string
	PurchaseOrderNumber string
	Status              string
	LineCount           int32
	AcceptedAt          *time.Time
	RejectedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	// Lines (populated only when the list request includes "lines").
	Lines []*DeliveryLine
}

// Delivery represents a full delivery with its lines.
type Delivery struct {
	ID                  string
	Number              string
	PurchaseOrderID     string
	PurchaseOrderNumber string
	Status              string
	Lines               []*DeliveryLine
	AcceptedAt          *time.Time
	RejectedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DeliveryLine represents a line item in a delivery.
type DeliveryLine struct {
	ID                        string
	ItemID                    *string
	ItemSKU                   *string
	ItemDescription           *string
	QuantityID                string
	QuantityValue             string
	QuantityUnitID            string
	QuantityUnitAbbreviation  string
	UnitCostID                string
	UnitCostValue             string
	UnitCostNumeratorUnitID   string
	UnitCostDenominatorUnitID string
	LocationID                *string
	LocationName              *string
	LotID                     *string
	LotNumber                 *string
	AcceptedAt                *time.Time
	RejectedAt                *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// ListDeliveriesParams holds parameters for listing deliveries.
type ListDeliveriesParams struct {
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

// ListDeliveriesResult holds the result of listing deliveries.
type ListDeliveriesResult struct {
	Deliveries []*DeliverySummary
	PageInfo   pagination.PageInfo
}

// GetDeliveryParams holds parameters for getting a single delivery.
type GetDeliveryParams struct {
	AccountID  string
	DeliveryID string
}
