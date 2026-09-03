package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
)

// PurchaseOrder represents a full purchase order domain model.
type PurchaseOrder struct {
	ID                    string
	Number                string                 `audit:"number"`
	Note                  *string                `audit:"note"`
	IsAcknowledgmentSent  bool                   `audit:"is_acknowledgment_sent"`
	BillingAddressID      string                 `audit:"billing_address_id"`
	ShippingAddressID     string                 `audit:"shipping_address_id"`
	CarrierID             *string                `audit:"carrier_id"`
	ServiceLevelID        *string                `audit:"service_level_id"`
	CarrierBillingType    *string                `audit:"carrier_billing_type"`
	CarrierBillingAccount *string                `audit:"carrier_billing_account"`
	PriorityCode          constants.PriorityCode `audit:"priority_code"`
	ShippingTermID        *string                `audit:"shipping_term_id"`
	SalesOrderStatusCode  string                 `audit:"sales_order_status_code"`
	SalesOrderTypeCode    string                 `audit:"sales_order_type_code"`
	PaymentTermID         *string                `audit:"payment_term_id"`
	BuyerAccountID        string                 `audit:"buyer_account_id"`
	SellerAccountID       string                 `audit:"seller_account_id"`
	OwnerAccountID        string
	IssuedAt              *time.Time `audit:"issued_at"`
	CompletedAt           *time.Time `audit:"completed_at"`
	PromisedAt            *time.Time `audit:"promised_at"`
	CreatedAt             time.Time
	UpdatedAt             time.Time

	// Joined fields for reads
	SupplierName   string
	SupplierNumber string
	StatusName     string
	TypeName       string
	PriorityName   string

	// Joined address details (same as sales order)
	BillToName        *string
	BillToIsDropShip  *bool
	BillToStreetLine1 *string
	BillToStreetLine2 *string
	BillToLocality    *string
	BillToState       *string
	BillToPostalCode  *string
	BillToCountry     *string
	BillToPhone       *string
	BillToEmail       *string
	BillToCreatedAt   *time.Time
	BillToUpdatedAt   *time.Time
	ShipToName        *string
	ShipToIsDropShip  *bool
	ShipToStreetLine1 *string
	ShipToStreetLine2 *string
	ShipToLocality    *string
	ShipToState       *string
	ShipToPostalCode  *string
	ShipToCountry     *string
	ShipToPhone       *string
	ShipToEmail       *string
	ShipToCreatedAt   *time.Time
	ShipToUpdatedAt   *time.Time

	// Joined carrier details
	CarrierName                 *string
	CarrierIsPortalEnabled      *bool
	CarrierCreatedAt            *time.Time
	CarrierUpdatedAt            *time.Time
	ServiceLevelName            *string
	ServiceLevelToken           *string
	ServiceLevelIsPortalEnabled *bool
	ServiceLevelCreatedAt       *time.Time
	ServiceLevelUpdatedAt       *time.Time

	// Joined payment/shipping term
	PaymentTermName             *string
	PaymentTermIsActive         *bool
	PaymentTermCreatedAt        *time.Time
	PaymentTermUpdatedAt        *time.Time
	ShippingTermName            *string
	ShippingTermIsFreightExempt *bool
	ShippingTermIsCarrierRate   *bool
	ShippingTermCreatedAt       *time.Time
	ShippingTermUpdatedAt       *time.Time

	// Joined priority
	PriorityID *string

	// Joined receiving order
	ReceivingOrderID *string
	Deliveries       []DocumentRef

	// Lines (populated when included)
	Lines []*PurchaseOrderLine

	// Contacts (populated when fetched)
	Contacts []*PurchaseOrderEmailContact

	// ReceivingOrder (populated when included)
	ReceivingOrder *ReceivingOrder
}

// PurchaseOrderSummary represents a purchase order for list views.
type PurchaseOrderSummary struct {
	ID                   string
	Number               string
	StatusCode           string
	StatusName           string
	TypeCode             string
	TypeName             string
	SupplierID           string
	SupplierName         string
	SupplierNumber       string
	LineCount            int32
	IsAcknowledgmentSent bool
	PriorityCode         constants.PriorityCode
	PriorityName         string
	PriorityID           *string
	IssuedAt             *time.Time
	CompletedAt          *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	// Lines (populated only when the list request includes "lines").
	Lines []*PurchaseOrderLine
}

// PurchaseOrderLine represents a purchase order line domain model.
type PurchaseOrderLine struct {
	ID                 string
	LineItemNumber     int32   `audit:"line_item_number"`
	ProductSKU         string  `audit:"product_sku"`
	ProductDescription *string `audit:"product_description"`
	ProductID          *string `audit:"product_id"`
	ItemID             *string `audit:"item_id"`
	ItemSKU            *string `audit:"item_sku"`
	SalesOrderID       string

	// Quantity ordered
	QuantityID               string
	QuantityValue            string `audit:"quantity_value"`
	QuantityUnitID           string `audit:"quantity_unit_id"`
	QuantityUnitName         string `audit:"quantity_unit_name"`
	QuantityUnitAbbreviation string `audit:"quantity_unit_abbreviation"`
	QuantityUnitType         string `audit:"quantity_unit_type"`

	// Quantity received
	QuantityReceivedValue *string `audit:"quantity_received_value"`

	// Unit price
	UnitPriceID                  string
	UnitPriceValue               string `audit:"unit_price_value"`
	UnitPriceNumeratorUnitID     string `audit:"unit_price_numerator_unit_id"`
	UnitPriceNumeratorUnitAbbr   string `audit:"unit_price_numerator_unit_abbr"`
	UnitPriceDenominatorUnitID   string `audit:"unit_price_denominator_unit_id"`
	UnitPriceDenominatorUnitAbbr string `audit:"unit_price_denominator_unit_abbr"`
	UnitPriceCreatedAt           time.Time
	UnitPriceUpdatedAt           time.Time

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

// PurchaseOrderEmailContact represents an email contact on a purchase order.
type PurchaseOrderEmailContact struct {
	ID            string
	AccountUserID string
}

// ListPurchaseOrdersParams holds the parameters for listing purchase orders.
type ListPurchaseOrdersParams struct {
	Cursor      *string
	Limit       int32
	Query       *string
	StatusCodes []string
	ItemIDs     []string
	SupplierIDs []string
	StartDate   *string
	EndDate     *string
	AccountID   string
	Includes    []string
}

// ListPurchaseOrdersResult holds the result of listing purchase orders.
type ListPurchaseOrdersResult struct {
	PurchaseOrders []*PurchaseOrderSummary
	PageInfo       pagination.PageInfo
}

// GetPurchaseOrderParams holds the parameters for getting a single purchase order.
type GetPurchaseOrderParams struct {
	PurchaseOrderID string
	AccountID       string
	Includes        []string
}

// CreatePurchaseOrderParams holds the parameters for creating a purchase order.
type CreatePurchaseOrderParams struct {
	AccountID             string
	SupplierAccountID     string
	Includes              []string
	Number                string
	SalesOrderStatusCode  string
	BillingAddressID      string
	ShippingAddressID     string
	Note                  *string
	CarrierID             *string
	ServiceLevelID        *string
	CarrierBillingType    *string
	CarrierBillingAccount *string
	PriorityCode          string
	ShippingTermID        *string
	PaymentTermID         *string
	PromisedAt            *string
	BillToName            *string
	BillToStreetLine1     *string
	BillToStreetLine2     *string
	BillToLocality        *string
	BillToState           *string
	BillToPostalCode      *string
	BillToCountry         *string
	ShipToName            *string
	ShipToStreetLine1     *string
	ShipToStreetLine2     *string
	ShipToLocality        *string
	ShipToState           *string
	ShipToPostalCode      *string
	ShipToCountry         *string
	Lines                 []CreatePurchaseOrderLineInput
	ContactAccountUserIDs []string
}

// CreatePurchaseOrderLineInput represents a line to create with a new purchase order.
type CreatePurchaseOrderLineInput struct {
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
}

// UpdatePurchaseOrderParams holds the parameters for updating a purchase order.
type UpdatePurchaseOrderParams struct {
	PurchaseOrderID       string
	AccountID             string
	Includes              []string
	Note                  *string
	Number                *string
	PriorityCode          *string
	BillingAddressID      *string
	ShippingAddressID     *string
	PromisedAt            *string
	ContactAccountUserIDs []string
}

// DeletePurchaseOrderParams holds the parameters for deleting a purchase order.
type DeletePurchaseOrderParams struct {
	PurchaseOrderID string
	AccountID       string
}

// BulkDeletePurchaseOrdersParams holds the parameters for bulk deleting purchase orders.
type BulkDeletePurchaseOrdersParams struct {
	PurchaseOrderIDs []string
	AccountID        string
}

// ChangePurchaseOrderStatusParams holds the parameters for changing a purchase order status.
type ChangePurchaseOrderStatusParams struct {
	PurchaseOrderID string
	AccountID       string
	StatusChange    string
	SendEmail       bool
	Includes        []string
}

// CreatePurchaseOrderLineParams holds the parameters for creating a purchase order line.
type CreatePurchaseOrderLineParams struct {
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
}

// UpdatePurchaseOrderLineParams holds the parameters for updating a purchase order line.
type UpdatePurchaseOrderLineParams struct {
	PurchaseOrderLineID        string
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
}

// DeletePurchaseOrderLineParams holds the parameters for deleting a purchase order line.
type DeletePurchaseOrderLineParams struct {
	PurchaseOrderLineID string
	SalesOrderID        string
	AccountID           string
}

// Purchase order status change constants.
const (
	PurchaseOrderStatusChangeIssue   = "issue"
	PurchaseOrderStatusChangeUnissue = "unissue"
	PurchaseOrderStatusChangeClose   = "close"
	PurchaseOrderStatusChangeOpen    = "open"
)
