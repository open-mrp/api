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

	// Joined fields for reads
	CustomerName             string
	CustomerNumber           string
	CustomerStatusCode       *string
	CustomerCommissionPolicy *string
	StatusName               string
	TypeName                 string
	PriorityName             string

	// Joined address details
	BillToName        *string
	BillToStreetLine1 *string
	BillToStreetLine2 *string
	BillToLocality    *string
	BillToState       *string
	BillToPostalCode  *string
	BillToCountry     *string
	BillToPhone       *string
	BillToEmail       *string
	ShipToName        *string
	ShipToStreetLine1 *string
	ShipToStreetLine2 *string
	ShipToLocality    *string
	ShipToState       *string
	ShipToPostalCode  *string
	ShipToCountry     *string
	ShipToPhone       *string
	ShipToEmail       *string

	// Joined carrier details
	CarrierName                 *string
	CarrierIsPortalEnabled      *bool
	ServiceLevelName            *string
	ServiceLevelToken           *string
	ServiceLevelIsPortalEnabled *bool

	// Joined sales rep
	SalesRepName *string

	// Joined payment/shipping term
	PaymentTermName             *string
	PaymentTermIsActive         *bool
	ShippingTermName            *string
	ShippingTermIsFreightExempt *bool
	ShippingTermIsCarrierRate   *bool

	// Joined order discount
	OrderDiscountName *string

	// Joined priority
	PriorityID *string

	// Joined pick
	PickID *string

	// Lines (populated when included)
	Lines []*SalesOrderLine
}

// SalesOrderSummary represents a sales order for list views.
type SalesOrderSummary struct {
	ID                       string
	Number                   string
	CustomerPONumber         *string
	StatusCode               string
	StatusName               string
	TypeCode                 string
	TypeName                 string
	CustomerID               string
	CustomerName             string
	CustomerNumber           string
	CustomerStatusCode       *string
	CustomerCommissionPolicy *string
	LineCount                int32
	IsAcknowledgmentSent     bool
	PriorityCode             constants.PriorityCode
	PriorityName             string
	PriorityID               *string
	IssuedAt                 *time.Time
	CompletedAt              *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// ListSalesOrdersParams holds the parameters for listing sales orders.
type ListSalesOrdersParams struct {
	Cursor                *string
	Limit                 int32
	Query                 *string
	StatusCodes           []string
	ItemIDs               []string
	ProductLineIDs        []string
	CustomerIDs           []string
	CustomerGroupIDs      []string
	SalesRepIDs           []string
	StartDate             *string
	EndDate               *string
	ExcludeInternalOrders bool
	AccountID             string
	BuyerAccountID        *string
}

// ListSalesOrdersResult holds the result of listing sales orders.
type ListSalesOrdersResult struct {
	SalesOrders []*SalesOrderSummary
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
	SalesOrderTypeCode    string
	PaymentTermID         *string
	OrderDiscountID       *string
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
	Lines                 []CreateSalesOrderLineInput
	// Email contact recipients to write into order_email_contact on create.
	AcknowledgementEmailContacts []SalesOrderEmailContactInput
	InvoiceEmailContacts         []SalesOrderEmailContactInput
}

// SalesOrderEmailContactInput represents a single recipient to wire to a sales order.
type SalesOrderEmailContactInput struct {
	AccountUserID string
}

// CreateSalesOrderLineInput represents a line to create with a new sales order.
type CreateSalesOrderLineInput struct {
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

// UpdateSalesOrderParams holds the parameters for updating a sales order.
type UpdateSalesOrderParams struct {
	SalesOrderID          string
	AccountID             string
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
	// When non-nil, replaces the acknowledgement email contacts on the order.
	// Empty slice clears all contacts; nil leaves existing contacts untouched.
	AcknowledgementEmailContacts *[]SalesOrderEmailContactInput
	// When non-nil, replaces the invoice email contacts on the order.
	// Empty slice clears all contacts; nil leaves existing contacts untouched.
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
