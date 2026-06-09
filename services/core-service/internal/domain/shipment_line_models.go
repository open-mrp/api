package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// ShipmentLine represents a shipment line domain model.
type ShipmentLine struct {
	ID               string
	ShipmentID       string
	SalesOrderLineID string  `audit:"sales_order_line_id"`
	OrderLineSKU     string  `audit:"order_line_sku"`
	OrderLineDesc    *string `audit:"order_line_desc"`
	OrderLineItemID  *string

	// Quantity
	QuantityID               string
	QuantityValue            string `audit:"quantity_value"`
	QuantityUnitID           string `audit:"quantity_unit_id"`
	QuantityUnitName         string `audit:"quantity_unit_name"`
	QuantityUnitAbbreviation string `audit:"quantity_unit_abbreviation"`
	QuantityUnitType         string `audit:"quantity_unit_type"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListShipmentLinesParams holds the parameters for listing shipment lines.
type ListShipmentLinesParams struct {
	AccountID  string
	ShipmentID string
	Cursor     *string
	Limit      int32
	Query      *string
}

// ListShipmentLinesResult holds the result of listing shipment lines.
type ListShipmentLinesResult struct {
	Lines    []*ShipmentLine
	PageInfo pagination.PageInfo
}

// CreateShipmentLineEndpointParams holds the parameters for creating a shipment line via the API.
type CreateShipmentLineEndpointParams struct {
	AccountID        string
	ShipmentID       string
	SalesOrderLineID string
	QuantityValue    string
	QuantityUnitID   string
}

// UpdateShipmentLineEndpointParams holds the parameters for updating a shipment line via the API.
type UpdateShipmentLineEndpointParams struct {
	AccountID      string
	ShipmentID     string
	ShipmentLineID string
	QuantityValue  *string
	QuantityUnitID *string
}

// DeleteShipmentLineEndpointParams holds the parameters for deleting a shipment line.
type DeleteShipmentLineEndpointParams struct {
	AccountID      string
	ShipmentID     string
	ShipmentLineID string
}
