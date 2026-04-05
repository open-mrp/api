package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

// Pick represents a pick domain model with joined fields for reads.
type Pick struct {
	ID           string
	Number       string `audit:"number"`
	SalesOrderID string
	AccountID    string
	FinishedAt   *time.Time `audit:"finished_at"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Joined fields for reads
	CustomerID     string
	CustomerName   string
	CustomerNumber string
	PriorityCode   constants.PriorityCode
	PriorityName   string

	// Populated conditionally
	Lines       []*PickLine
	Departments []*PickDepartment
}

// PickSummary represents a pick for list views.
type PickSummary struct {
	ID             string
	Number         string
	SalesOrderID   string
	CustomerID     string
	CustomerName   string
	CustomerNumber string
	PriorityCode   constants.PriorityCode
	PriorityName   string
	FinishedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PickDepartment represents a department associated with a pick.
type PickDepartment struct {
	ID   string
	Name string
}

// PickLine represents a pick line domain model with joined fields.
type PickLine struct {
	ID                       string
	PickID                   string
	SalesOrderLineID         string
	QuantityID               string
	QuantityValue            string `audit:"quantity_value"`
	QuantityUnitID           string
	QuantityUnitName         string
	QuantityUnitAbbreviation string
	PackedAt                 *time.Time `audit:"packed_at"`
	CreatedAt                time.Time
	UpdatedAt                time.Time

	// Joined order line info
	OrderLineItemNumber       int32
	OrderLineSKU              string
	OrderLineDescription      *string
	OrderedQuantityValue      string
	OrderedQuantityUnitID     string
	OrderedQuantityUnitName   string
	OrderedQuantityUnitAbbrev string
}

// ListPicksParams holds the parameters for listing picks.
type ListPicksParams struct {
	AccountID        string
	Cursor           *string
	Limit            int32
	Query            *string
	Status           *string // "open", "closed", or nil for all
	CustomerIDs      []string
	ProductLineIDs   []string
	CustomerGroupIDs []string
	DepartmentIDs    []string
	StartDate        *string
	EndDate          *string
}

// ListPicksResult holds the result of listing picks.
type ListPicksResult struct {
	Picks    []*PickSummary
	PageInfo pagination.PageInfo
}

// UpdatePickParams holds the parameters for updating a pick.
type UpdatePickParams struct {
	AccountID  string
	PickID     string
	Number     *string
	FinishedAt **time.Time // double pointer: nil = not provided, *nil = set to null
}

// UpdatePickLineParams holds the parameters for updating a pick line's quantity.
type UpdatePickLineParams struct {
	AccountID     string
	PickID        string
	PickLineID    string
	QuantityValue *string
}

// GetPickShipmentsParams holds the parameters for getting shipment numbers for a pick.
type GetPickShipmentsParams struct {
	AccountID string
	PickID    string
	Query     *string
	Limit     int32
	Offset    int32
}

// PickShipmentsResult holds the result of getting shipment numbers for a pick.
type PickShipmentsResult struct {
	ShipmentNumbers []string
	Count           int32
}

// PackPickResult holds the result of packing a pick.
type PackPickResult struct {
	Pick           *Pick
	ShipmentNumber string
}

// PickSalesOrder holds order info needed for shipment creation during pack.
type PickSalesOrder struct {
	ID                string
	Number            string
	CarrierID         string
	ServiceLevelID    *string
	ShippingAddressID string
}

// CreateShipmentFromPickParams holds the parameters for creating a shipment during pack.
type CreateShipmentFromPickParams struct {
	ID                string
	Number            string
	SalesOrderID      string
	CarrierID         string
	ServiceLevelID    *string
	ShippingAddressID string
	StatusCode        string
	AccountID         string
}

// CreateShipmentLineParams holds the parameters for creating a shipment line during pack.
type CreateShipmentLineParams struct {
	ID               string
	ShipmentID       string
	SalesOrderLineID string
	QuantityID       string
}

// CreateShippingCaseParams holds the parameters for creating a shipping case during pack.
type CreateShippingCaseParams struct {
	ID              string
	Number          string
	FreightAmountID string
	FreightWeightID string
	ShipmentID      string
	CarrierID       string
	AccountID       string
}
