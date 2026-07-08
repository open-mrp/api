package domain

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

// SalesOrder represents a full sales order domain model.
type SalesOrder struct {
	ID                    string
	Number                string                 `audit:"number"`
	CustomerPONumber      *string                `audit:"customer_po_number"`
	Note                  *string                `audit:"note"`
	IsAcknowledgmentSent  bool                   `audit:"is_acknowledgment_sent"`
	BillingAddressID      string                 `audit:"billing_address_id"`
	ShippingAddressID     string                 `audit:"shipping_address_id"`
	CarrierID             *string                `audit:"carrier_id"`
	ServiceLevelID        *string                `audit:"service_level_id"`
	CarrierBillingType    *string                `audit:"carrier_billing_type"`
	CarrierBillingAccount *string                `audit:"carrier_billing_account"`
	PriorityCode          constants.PriorityCode `audit:"priority_code"`
	SalesRepID            *string                `audit:"sales_rep_id"`
	ShippingTermID        *string                `audit:"shipping_term_id"`
	SalesOrderStatusCode  string                 `audit:"sales_order_status_code"`
	SalesOrderTypeCode    string                 `audit:"sales_order_type_code"`
	PaymentTermID         *string                `audit:"payment_term_id"`
	ProductionRunID       *string                `audit:"production_run_id"`
	OrderDiscountID       *string                `audit:"order_discount_id"`
	BuyerAccountID        string                 `audit:"buyer_account_id"`
	SellerAccountID       string                 `audit:"seller_account_id"`
	OwnerAccountID        string                 `audit:"owner_account_id"`
	IssuedAt              *time.Time             `audit:"issued_at"`
	CompletedAt           *time.Time             `audit:"completed_at"`
	FirstShipAt           *time.Time             `audit:"first_ship_at"`
	ExpiredAt             *time.Time             `audit:"expired_at"`
	PromisedAt            *time.Time             `audit:"promised_at"`
	CreatedAt             time.Time
	UpdatedAt             time.Time

	// Joined customer details
	CustomerName             string
	CustomerNumber           string
	CustomerStatusCode       *string
	CustomerCommissionPolicy *string
	CustomerCreatedAt        *time.Time
	CustomerUpdatedAt        *time.Time
	StatusName               string
	TypeName                 string
	PriorityName             string

	// Joined address details
	BillToName          *string
	BillToIsDropShip    *bool
	BillToGeolocationID *string
	BillToStreetLine1   *string
	BillToStreetLine2   *string
	BillToLocality      *string
	BillToState         *string
	BillToPostalCode    *string
	BillToCountry       *string
	BillToPhone         *string
	BillToEmail         *string
	BillToCreatedAt     *time.Time
	BillToUpdatedAt     *time.Time
	ShipToName          *string
	ShipToIsDropShip    *bool
	ShipToGeolocationID *string
	ShipToStreetLine1   *string
	ShipToStreetLine2   *string
	ShipToLocality      *string
	ShipToState         *string
	ShipToPostalCode    *string
	ShipToCountry       *string
	ShipToPhone         *string
	ShipToEmail         *string
	ShipToCreatedAt     *time.Time
	ShipToUpdatedAt     *time.Time

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

	// Joined sales rep
	SalesRepName *string

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

	// Joined order discount
	OrderDiscountName         *string
	OrderDiscountCode         *string
	OrderDiscountPercentage   *string
	OrderDiscountAmount       *string
	OrderDiscountDiscountType *string
	OrderDiscountOrderCount   *int32
	OrderDiscountCreatedAt    *time.Time
	OrderDiscountUpdatedAt    *time.Time

	// Joined priority
	PriorityID *string

	// Joined pick
	PickID *string

	// Count of order lines (always populated, independent of the lines include).
	LineCount int32

	// Derived payment state (always populated): computed from settlement allocations vs. invoiced amounts plus any Stripe payment intent.
	PaymentStatus constants.SalesOrderPaymentStatus

	// Stripe payment intent IDs recorded against this order (always populated).
	PaymentIntentIDs []string

	// Lines (populated when included)
	Lines []*SalesOrderLine

	// IDs of shipments linked to this order (populated when related.shipments is included).
	ShipmentIDs []string

	// Invoice email recipients (populated when contacts is included).
	InvoiceEmails []string

	// Order acknowledgement email recipients (populated when contacts is included).
	AcknowledgementEmails []string
}

// SalesOrderContacts holds a sales order's email recipients grouped by notification type.
type SalesOrderContacts struct {
	InvoiceEmails         []string
	AcknowledgementEmails []string
}

// ListSalesOrdersParams holds the parameters for listing sales orders.
type ListSalesOrdersParams struct {
	Cursor           *string
	Limit            int32
	Query            *string
	StatusCodes      []string
	ItemIDs          []string
	ProductLineIDs   []string
	CustomerIDs      []string
	CustomerGroupIDs []string
	SalesRepIDs      []string
	StartDate        *string
	EndDate          *string
	AccountID        string
	BuyerAccountID   *string
	// Includes to expand (e.g. "lines"); inline-joined fields are always present.
	Includes []string
}

// ListSalesOrdersResult holds the result of listing sales orders.
type ListSalesOrdersResult struct {
	SalesOrders []*SalesOrder
	PageInfo    pagination.PageInfo
}

// GetSalesOrderParams holds the parameters for getting a single sales order.
type GetSalesOrderParams struct {
	SalesOrderID   string
	AccountID      string
	BuyerAccountID *string
	Includes       []string
}

// CreateSalesOrderParams holds the parameters for creating a sales order.
type CreateSalesOrderParams struct {
	AccountID             string
	BuyerAccountID        string
	Includes              []string
	SellerAccountID       string
	OwnerAccountID        string
	Number                string
	SalesOrderStatusCode  string
	BillingAddressID      string
	ShippingAddressID     string
	CustomerPONumber      *string
	Note                  *string
	CarrierID             *string
	ServiceLevelID        *string
	CarrierBillingType    *string
	CarrierBillingAccount *string
	PriorityCode          string
	SalesRepID            *string
	ShippingTermID        *string
	PaymentTermID         *string
	OrderDiscountID       *string
	PromisedAt            *time.Time
	// Existing bill-to / ship-to address IDs the order references. The addresses must belong to the order's owner or buyer account (matching Dashboard, which only accepts address IDs — addresses are persisted separately).
	BillToAddressID string
	ShipToAddressID string
	Lines           []CreateSalesOrderLineInput
	// Email contact recipients to write into order_email_contact on create.
	AcknowledgementEmailContacts []SalesOrderEmailContactInput
	InvoiceEmailContacts         []SalesOrderEmailContactInput
}

// SalesOrderEmailContactInput represents a single recipient to wire to a sales order.
type SalesOrderEmailContactInput struct {
	AccountUserID string
}

// ProductTypeLine carries a product's type code and product line ID, used by shipping-rate estimation (parcel weight + product-line freight exemption) on create.
type ProductTypeLine struct {
	ProductID       string
	ProductTypeCode string
	ProductLineID   *string
}

// CreateSalesOrderLineInput represents a line to create with a new sales order. The item, SKU/description defaults, unit cost, and (unless an internal user overrides) the unit price are all resolved server-side from the product.
type CreateSalesOrderLineInput struct {
	ProductID      string
	QuantityValue  string
	QuantityUnitID string
	// ProductSKU / ProductDescription default to the product's when nil.
	ProductSKU         *string
	ProductDescription *string
	// UnitPrice is an optional override, honored only for internal actors.
	UnitPrice *RateValue
}

// UpdateSalesOrderParams holds the parameters for updating a sales order.
type UpdateSalesOrderParams struct {
	SalesOrderID          string
	AccountID             string
	Includes              []string
	Number                *string
	BillingAddressID      *string
	ShippingAddressID     *string
	CustomerPONumber      *string
	Note                  *string
	CarrierID             *string
	ServiceLevelID        *string
	CarrierBillingType    *string
	CarrierBillingAccount *string
	PriorityCode          *string
	SalesRepID            *string
	ShippingTermID        *string
	PaymentTermID         *string
	OrderDiscountID       *string
	IsAcknowledgmentSent  *bool
	PromisedAt            *time.Time
	BuyerAccountID        *string
	// When non-nil, replaces the acknowledgement email contacts on the order. Empty slice clears all contacts; nil leaves existing contacts untouched.
	AcknowledgementEmailContacts *[]SalesOrderEmailContactInput
	// When non-nil, replaces the invoice email contacts on the order. Empty slice clears all contacts; nil leaves existing contacts untouched.
	InvoiceEmailContacts *[]SalesOrderEmailContactInput
}

// DeleteSalesOrderParams holds the parameters for deleting a sales order.
type DeleteSalesOrderParams struct {
	SalesOrderID string
	AccountID    string
}

// BulkDeleteSalesOrdersParams holds the parameters for bulk deleting sales orders.
type BulkDeleteSalesOrdersParams struct {
	SalesOrderIDs []string
	AccountID     string
}

// ChangeSalesOrderStatusParams holds the parameters for changing a sales order status.
type ChangeSalesOrderStatusParams struct {
	SalesOrderID string
	AccountID    string
	StatusChange string
	SendEmail    bool
	Includes     []string
}

// CheckoutSalesOrderParams holds the parameters for checking out a sales order.
type CheckoutSalesOrderParams struct {
	SalesOrderID string
	AccountID    string
	Email        string
	SuccessURL   *string
	CancelURL    *string
}

// CheckoutSalesOrderResult holds the result of a checkout operation.
type CheckoutSalesOrderResult struct {
	CheckoutURL string
}

// CreateSalesOrderProductionRunParams holds the parameters for creating a production run from a sales order.
type CreateSalesOrderProductionRunParams struct {
	SalesOrderID string
	AccountID    string
}

// CreateSalesOrderProductionRunResult holds the result of creating a production run.
type CreateSalesOrderProductionRunResult struct {
	ProductionRunID string
}

// SalesOrderLineForBOM represents a sales order line with BOM-relevant fields.
type SalesOrderLineForBOM struct {
	ID             string
	ItemID         string
	QuantityValue  decimal.Decimal
	QuantityUnitID string
}

// SalesOrderSaleLineForIssue represents a sale-type order line used during issue operations.
type SalesOrderSaleLineForIssue struct {
	ID             string
	ItemID         *string
	QuantityValue  string
	QuantityUnitID string
}
