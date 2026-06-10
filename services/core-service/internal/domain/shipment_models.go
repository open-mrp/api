package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// Shipment represents the full detail of a shipment domain model.
type Shipment struct {
	ID                           string
	Number                       string     `audit:"number"`
	Note                         *string    `audit:"note"`
	BillOfLading                 *string    `audit:"bill_of_lading"`
	MasterTrackingNumber         *string    `audit:"master_tracking_number"`
	StatusCode                   string     `audit:"status_code"`
	StatusName                   string     `audit:"status_name"`
	ShippedAt                    *time.Time `audit:"shipped_at"`
	SalesOrderID                 string
	SalesOrderNumber             string
	CustomerPONumber             *string
	CarrierBillingType           *string `audit:"carrier_billing_type"`
	CarrierBillingAccount        *string `audit:"carrier_billing_account"`
	CustomerID                   string
	CustomerName                 string
	CustomerNumber               string
	CustomerStatusCode           *string
	CustomerCommissionPolicy     *string
	CustomerCreatedAt            time.Time
	CustomerUpdatedAt            time.Time
	CarrierID                    string `audit:"carrier_id"`
	CarrierName                  string `audit:"carrier_name"`
	CarrierIsPortalEnabled       *bool
	CarrierCreatedAt             *time.Time
	CarrierUpdatedAt             *time.Time
	ServiceLevelID               *string `audit:"service_level_id"`
	ServiceLevelName             *string `audit:"service_level_name"`
	ServiceLevelToken            *string
	ServiceLevelIsPortalEnabled  *bool
	ServiceLevelCreatedAt        *time.Time
	ServiceLevelUpdatedAt        *time.Time
	ShippingAddressID            string
	ShippingAddressName          *string
	ShippingAddressPhone         *string
	ShippingAddressEmail         *string
	ShippingAddressIsDropShip    *bool
	ShippingAddressGeolocationID *string
	ShippingAddressStreetLine1   *string
	ShippingAddressStreetLine2   *string
	ShippingAddressLocality      *string
	ShippingAddressState         *string
	ShippingAddressPostalCode    *string
	ShippingAddressCountry       *string
	ShippingAddressCreatedAt     *time.Time
	ShippingAddressUpdatedAt     *time.Time
	ShippedByID                  *string `audit:"shipped_by_id"`
	ShippedByName                *string `audit:"shipped_by_name"`
	ShippedByStatusCode          *string
	ShippedByCreatedAt           *time.Time
	ShippedByUpdatedAt           *time.Time
	InvoiceID                    *string
	InvoiceNumber                *string
	InvoiceCreatedAt             *time.Time
	InvoiceUpdatedAt             *time.Time
	PickID                       *string
	PickNumber                   *string
	PickCreatedAt                *time.Time
	PickUpdatedAt                *time.Time
	SalesOrderCreatedAt          time.Time
	SalesOrderUpdatedAt          time.Time
	BillingAddressCountry        *string
	BillingAddressZip            *string
	AccountID                    string
	CreatedAt                    time.Time
	UpdatedAt                    time.Time

	// Expandable collections
	Lines         []*ShipmentLine
	ShippingCases []*ShippingCase
}

// ShipmentSummary represents a shipment in list views.
type ShipmentSummary struct {
	ID                          string
	Number                      string
	Note                        *string
	BillOfLading                *string
	MasterTrackingNumber        *string
	StatusCode                  string
	StatusName                  string
	ShippedAt                   *time.Time
	SalesOrderID                string
	SalesOrderNumber            string
	SalesOrderCreatedAt         time.Time
	SalesOrderUpdatedAt         time.Time
	CustomerID                  string
	CustomerName                string
	CustomerNumber              string
	CustomerStatusCode          *string
	CustomerCommissionPolicy    *string
	CustomerCreatedAt           time.Time
	CustomerUpdatedAt           time.Time
	CarrierID                   string
	CarrierName                 string
	CarrierIsPortalEnabled      *bool
	ServiceLevelID              *string
	ServiceLevelName            *string
	ServiceLevelToken           *string
	ServiceLevelIsPortalEnabled *bool
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
	// Lines (populated only when the list request includes "lines").
	Lines []*ShipmentLine
}

// ListShipmentsParams holds the parameters for listing shipments.
type ListShipmentsParams struct {
	AccountID        string
	Cursor           *string
	Limit            int32
	Query            *string
	Status           *string
	ItemIDs          []string
	CustomerIDs      []string
	ProductLineIDs   []string
	CustomerGroupIDs []string
	SalesRepIDs      []string
	StartDate        *string
	EndDate          *string
	Includes         []string
}

// ListShipmentsResult holds the result of listing shipments.
type ListShipmentsResult struct {
	Shipments []*ShipmentSummary
	PageInfo  pagination.PageInfo
}

// GetShipmentParams holds the parameters for getting a shipment.
type GetShipmentParams struct {
	AccountID  string
	ShipmentID string
	Includes   []string
}

// UpdateShipmentParams holds the parameters for updating a shipment.
type UpdateShipmentParams struct {
	AccountID            string
	ShipmentID           string
	Note                 *string
	Number               *string
	MasterTrackingNumber *string
	CarrierID            *string
	ServiceLevelID       *string
	Includes             []string
}

// DeleteShipmentParams holds the parameters for deleting a shipment.
type DeleteShipmentParams struct {
	AccountID  string
	ShipmentID string
}

// ShipShipmentParams holds the parameters for shipping a shipment.
type ShipShipmentParams struct {
	AccountID     string
	ShipmentID    string
	EmailCustomer bool
	Includes      []string
}

// VoidShipmentParams holds the parameters for voiding a shipment.
type VoidShipmentParams struct {
	AccountID  string
	ShipmentID string
}
