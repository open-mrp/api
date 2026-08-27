package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
)

// Pick represents a pick domain model with joined fields for reads.
type Pick struct {
	ID               string
	Number           string `audit:"number"`
	SalesOrderID     string
	SalesOrderNumber string
	AccountID        string
	FinishedAt       *time.Time `audit:"finished_at"`
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Joined fields for reads
	CustomerID     string
	CustomerName   string
	CustomerNumber string
	PriorityID     string
	PriorityCode   constants.PriorityCode
	PriorityName   string

	// Server-computed roll-ups, so a row can show counts and progress without expanding its lines.
	LineCount        int32
	LastShippedAt    *time.Time
	PickedCompletion float64
	PackedCompletion float64

	// Ship-to carried from the sales order, so a pick renders its header without fetching the order.
	PromisedAt *time.Time
	// The order's cross-reference and instructions, carried so the floor works the pick without opening the order.
	CustomerPONumber *string
	Note             *string
	// Freight carried from the sales order, so a pick shows the carrier it ships on.
	CarrierID                   *string
	CarrierName                 *string
	CarrierIsPortalEnabled      *bool
	CarrierCreatedAt            *time.Time
	CarrierUpdatedAt            *time.Time
	ServiceLevelID              *string
	ServiceLevelName            *string
	ServiceLevelIsPortalEnabled *bool
	ServiceLevelToken           *string
	ServiceLevelCreatedAt       *time.Time
	ServiceLevelUpdatedAt       *time.Time
	CarrierBillingType          *string
	CarrierBillingAccount       *string
	// The order's delivery commitment and the rules that produced it, carried so a pick can explain
	// its dates without fetching the order.
	ShipByDate                 *time.Time
	LeadTimeDays               *int32
	LeadTimeSource             *constants.LeadTimeSource
	TransitDays                *int32
	TransitSource              *constants.TransitSource
	ShippingAddressID          string
	ShippingAddressName        *string
	ShippingAddressPhone       *string
	ShippingAddressEmail       *string
	ShippingAddressIsDropShip  *bool
	ShippingAddressGeolocation *string
	ShippingAddressStreetLine1 *string
	ShippingAddressStreetLine2 *string
	ShippingAddressLocality    *string
	ShippingAddressState       *string
	ShippingAddressPostalCode  *string
	ShippingAddressCountry     *string

	// Populated conditionally
	Lines       []*PickLine
	Departments []*PickDepartment
	ShipmentIDs []string
}

// Carries the picked/packed completion fractions for one pick, aggregated over its sale lines.
type PickProgress struct {
	PickedCompletion float64
	PackedCompletion float64
}

// PickDepartment represents a department associated with a pick.
type PickDepartment struct {
	ID   string
	Name string
}

// PickLine represents a pick line domain model with joined fields.
type PickLine struct {
	// OrderLineItemID is the item on the originating order line, so lines.item resolves.
	OrderLineItemID          *string
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
	OrderLineProductID        *string
	OrderedQuantityID         string
	OrderedQuantityValue      string
	OrderedQuantityUnitID     string
	OrderedQuantityUnitName   string
	OrderedQuantityUnitAbbrev string

	UnitPriceID                          string
	UnitPriceValue                       string
	UnitPriceNumeratorUnitID             string
	UnitPriceNumeratorUnitAbbreviation   string
	UnitPriceDenominatorUnitID           string
	UnitPriceDenominatorUnitAbbreviation string
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
	Includes         []string
	// Empty means the default, soonest ship-by date first.
	Sort constants.PickSort
}

// ListPicksResult holds the result of listing picks.
type ListPicksResult struct {
	Picks    []*Pick
	PageInfo pagination.PageInfo
}

// UpdatePickParams holds the parameters for updating a pick.
type UpdatePickParams struct {
	AccountID  string
	PickID     string
	Number     *string
	FinishedAt **time.Time // double pointer: nil = not provided, *nil = set to null
	Includes   []string
}

// Carries the parameters for updating a pick line's picked quantity.
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

// Records what a pack was accepted to do. Stored on the job, so it carries only resolved ids.
type PackPickJob struct {
	PickID            string
	ShipmentCaseCount int32
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
