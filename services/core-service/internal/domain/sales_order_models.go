package domain

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/metadata"
	"github.com/open-mrp/apikit/pagination"
)

// SalesOrder represents a full sales order domain model.
type SalesOrder struct {
	ID               string
	Number           string  `audit:"number"`
	CustomerPONumber *string `audit:"customer_po_number"`
	// Instructions is the reading party's own guidance for the order (document_party), internal to that party.
	Instructions          *string                `audit:"instructions"`
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
	ShipByDate            *time.Time             `audit:"ship_by_date"`
	LeadTimeDays          *int32                 `audit:"lead_time_days"`
	LeadTimeSourceCode    *string                `audit:"lead_time_source_code"`
	TransitDays           *int32                 `audit:"transit_days"`
	TransitSourceCode     *string                `audit:"transit_source_code"`
	// LeadTimeOverrideDays and ShipByOverrideDate are the two per-order commitment bases beside PromisedAt. At most one of the three is ever set; the write path rejects more.
	LeadTimeOverrideDays *int       `audit:"lead_time_override_days"`
	ShipByOverrideDate   *time.Time `audit:"ship_by_override_date"`
	// ShipByCutoffAt is ShipByDate at the plant's pickup cutoff, and CalendarAdjustmentDays how many days the receiving and shipping calendars pulled the date back beyond transit. Both stamped with the commitment so an edited calendar cannot rewrite the explanation of a promise already made.
	ShipByCutoffAt         *time.Time `audit:"ship_by_cutoff_at"`
	CalendarAdjustmentDays *int32     `audit:"calendar_adjustment_days"`
	// PromisedLeadTimeDays is the lead time the ship-by date was promised on, in business days, and
	// PromisedLeadTimeSource where it came from (a constants.PromisedLeadTimeSource). Both nil when the order was
	// promised or pinned a date rather than a lead time, or was issued before they were recorded.
	PromisedLeadTimeDays   *int32  `audit:"promised_lead_time_days"`
	PromisedLeadTimeSource *string `audit:"promised_lead_time_source"`
	// CommitmentFirmness is a constants.CommitmentFirmness, nil when unset.
	CommitmentFirmness *string `audit:"commitment_firmness"`
	// BillTo and ShipTo are the order's own copies of its addresses.
	BillTo *AddressSnapshot `audit:"bill_to"`
	ShipTo *AddressSnapshot `audit:"ship_to"`
	// SiteID is the site the order ships from; nil is the account's default site.
	SiteID    *string `audit:"site_id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	// Metadata is the reading party's own key/value pairs (document_party); never nil. See shared/metadata.
	Metadata map[string]string `audit:"metadata"`

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

	InvoiceIDs []string

	// Invoice email recipients (populated when contacts is included).
	InvoiceEmails []string

	// Order acknowledgement email recipients (populated when contacts is included).
	AcknowledgementEmails []string

	// EmailRecipients are the order's recipients and the documents each receives (populated when email_recipients is
	// included).
	EmailRecipients []NotificationRecipient

	// Fulfillment progress as fractions between 0 and 1 aggregated over the order's sale-type lines (always populated, independent of the lines include).
	PickedCompletion   float64
	PackedCompletion   float64
	InvoicedCompletion float64

	// Monetary stage totals summed over the order's lines, populated only when the totals include is requested (nil otherwise). Computed in Go via shared/pricing so the amounts match the dashboard exactly.
	Totals *SalesOrderTotals
}

// SalesOrderReferences blocks deleting a sales order once a shipment or invoice exists for it.
var SalesOrderReferences = RegisterReferences(References{
	Noun: "sales order",
	Probes: []ReferenceProbe{
		{Label: "shipments", Table: "shipment", Column: "sales_order_id"},
		{Label: "invoices", Table: "invoice", Column: "sales_order_id"},
	},
})

// SalesOrderTotals holds a sales order's monetary stage totals as decimal strings. Available is false when a line cannot be priced the way the dashboard prices it, in which case the amounts are empty and callers serialize a null total rather than a wrong one.
type SalesOrderTotals struct {
	Available bool
	Ordered   string
	Picked    string
	Packed    string
	Invoiced  string
}

// SalesOrderFulfillmentProgress holds an order's picked/packed/invoiced completion fractions (0..1), aggregated over its sale-type lines.
type SalesOrderFulfillmentProgress struct {
	PickedCompletion   float64
	PackedCompletion   float64
	InvoicedCompletion float64
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
	ShipByAfter      *string
	ShipByBefore     *string
	PastDue          *bool
	// CreatedAfter (inclusive) and CreatedBefore (exclusive) bound the created date.
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	// PaymentStatuses are constants.SalesOrderPaymentStatus values.
	PaymentStatuses []string
	AccountID       string
	BuyerAccountID  *string
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
	AccountID            string
	BuyerAccountID       string
	Includes             []string
	SellerAccountID      string
	OwnerAccountID       string
	Number               string
	SalesOrderStatusCode string
	BillingAddressID     string
	ShippingAddressID    string
	CustomerPONumber     *string
	// Instructions set or cleared by the caller; unset copies the customer's.
	Instructions          field.Clearable[string]
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
	// LeadTimeOverrideDays and ShipByOverrideDate are the alternatives to PromisedAt. At most one of the three may be set; the service rejects more.
	LeadTimeOverrideDays *int32
	ShipByOverrideDate   *time.Time
	// BillTo and ShipTo name a saved address (AddressID) or give one inline; nil takes the customer's default.
	BillTo *AddressSnapshot
	ShipTo *AddressSnapshot
	Lines  []CreateSalesOrderLineInput
	// EmailRecipients, when set (even empty), are the order's recipients; unset copies the customer's.
	EmailRecipients *[]NotificationRecipientRef
	// SiteID is the site the order ships from; nil takes the account's default site.
	SiteID *string
	// CommitmentFirmness is a constants.CommitmentFirmness, nil when unset.
	CommitmentFirmness *string
	Metadata           map[string]string
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
	// Overrides the server-computed price; honored only for internal actors.
	UnitPrice *RateValue
	Metadata  map[string]string
}

// UpdateSalesOrderParams holds the parameters for updating a sales order.
type UpdateSalesOrderParams struct {
	SalesOrderID string
	AccountID    string
	Includes     []string
	// Optional (set-or-leave; non-nullable — cannot be cleared).
	Number               *string
	BillingAddressID     *string
	ShippingAddressID    *string
	SiteID               *string
	CarrierID            *string
	PriorityCode         *string
	ShippingTermID       *string
	PaymentTermID        *string
	IsAcknowledgmentSent *bool
	BuyerAccountID       *string
	// Clearable (set / clear / leave). The service backfills unset from the existing
	// order, so a cleared field resolves to NULL and an unset field keeps its value.
	CustomerPONumber      field.Clearable[string]
	Instructions          field.Clearable[string]
	CommitmentFirmness    field.Clearable[string]
	ServiceLevelID        field.Clearable[string]
	CarrierBillingType    field.Clearable[string]
	CarrierBillingAccount field.Clearable[string]
	SalesRepID            field.Clearable[string]
	OrderDiscountID       field.Clearable[string]
	PromisedAt            field.Clearable[time.Time]
	// LeadTimeOverrideDays and ShipByOverrideDate are the alternatives to PromisedAt. Clearing one to set another in the same request is how a caller switches basis.
	LeadTimeOverrideDays field.Clearable[int32]
	ShipByOverrideDate   field.Clearable[time.Time]
	// When non-nil, replaces the order's email recipients. An empty slice clears them; nil leaves them untouched.
	EmailRecipients *[]NotificationRecipientRef
	// When non-nil, replace the order's recipients of acknowledgments or invoices alone.
	AcknowledgementEmailContacts *[]SalesOrderEmailContactInput
	InvoiceEmailContacts         *[]SalesOrderEmailContactInput
	// BillTo and ShipTo replace the order's copy of an address: a saved address (AddressID) or one given inline.
	BillTo   *AddressSnapshot
	ShipTo   *AddressSnapshot
	Metadata metadata.Update

	// Set by the service: the resolved copies of BillTo and ShipTo, written with the update.
	BillToSnapshot *AddressSnapshot
	ShipToSnapshot *AddressSnapshot
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
	ProductionRun *ProductionRun
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

// SalesOrderItemRemainder is the part of an item's ordered quantity the order has not yet issued or reserved, in UnitID.
type SalesOrderItemRemainder struct {
	ItemID         string
	UnitID         string
	RemainingValue string
}

// CustomerLeadTimeChain is what a buyer's ship-by commitment can be resolved from, every level together.
//
// All of them are returned rather than only the winner because the source is stamped onto the order beside the date: an order has to be able to say which rule produced its commitment, not just what the commitment was.
type CustomerLeadTimeChain struct {
	AccountRelationID string
	AccountGroupID    *string
	// ParentCustomerAccountID is the buyer's parent account, set whether or not the parent carries a lead time.
	ParentCustomerAccountID    *string
	CustomerLeadTimeDays       *int
	ParentCustomerLeadTimeDays *int
	AccountGroupLeadTimeDays   *int
}

// ShipByCommitment is the resolved promise stamped onto an order at issue.
type ShipByCommitment struct {
	ShipByDate   time.Time
	LeadTimeDays int
	SourceCode   string
	// PromisedLeadTimeDays is the lead time promised, in business days, and PromisedLeadTimeSource where it came from
	// (a constants.PromisedLeadTimeSource). Nil and empty when the order was promised or pinned a date.
	PromisedLeadTimeDays   *int
	PromisedLeadTimeSource string
	// TransitDays is the carrier transit subtracted from a promised delivery date to get ShipByDate, nil when transit was unknown or the commitment came from a lead time.
	TransitDays *int
	// TransitSourceCode names where TransitDays came from, empty when TransitDays is nil.
	TransitSourceCode string
	// ShipByCutoffAt is ShipByDate at the plant's pickup cutoff, nil when the ship calendar carries no cutoff.
	ShipByCutoffAt *time.Time
	// CalendarAdjustmentDays is how many days the receiving and shipping calendars pulled ShipByDate back beyond transit.
	CalendarAdjustmentDays int
	EstimatedDeliveryDate  *time.Time
	// Steps is the ordered derivation, one entry per rule that touched the date. Computed rather than stored: it explains a commitment to whoever is looking at it now, where the stamped scalars are what the commitment itself is made of.
	Steps []CommitmentStep
}

// CommitmentStep is one rule's contribution to a ship-by date.
type CommitmentStep struct {
	Code      string
	Date      time.Time
	DaysMoved int
	Detail    string
}

// QuoteCommitmentParams previews what a set of inputs would commit to, without writing anything.
//
// SalesOrderID and the loose fields are alternatives: an existing order supplies its own customer, address and carrier, while the order-entry form has none of that saved yet and passes them directly. The bases apply either way, so a form can preview a change to an order it has not saved.
type QuoteCommitmentParams struct {
	AccountID    string
	SalesOrderID *string

	BuyerAccountID  *string
	ShipToAddressID *string
	CarrierID       *string
	ServiceLevelID  *string
	IssuedAt        *time.Time

	PromisedAt           *time.Time
	LeadTimeOverrideDays *int32
	ShipByOverrideDate   *time.Time
}
