package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

// Customer represents a full customer record from the database.
type Customer struct {
	ID     string
	Name   string `audit:"name"`
	Number string `audit:"number"`
	// Status is the customer's credit standing: normal, hold_shipment or hold_all.
	Status constants.AccountStatusCode `audit:"standing"`
	// ArchiveStatus is whether the customer can receive new orders.
	ArchiveStatus       constants.ArchiveStatus    `audit:"status"`
	IsParentAccount     bool                       `audit:"is_parent_account"`
	CommissionPolicy    constants.CommissionPolicy `audit:"commission_policy"`
	FreightPolicy       constants.FreightPolicy    `audit:"freight_policy"`
	DefaultLeadTimeDays *int32                     `audit:"default_lead_time_days"`
	// ReceiveCalendarID is the days this customer's dock accepts freight, in the same chain as the lead time above.
	ReceiveCalendarID *string `audit:"receive_calendar_id"`
	// Supply is where this customer's orders are supplied from when the order line doesn't say: stock, make or buy. Nil falls through to the item chain.
	Supply *constants.Supply `audit:"supply"`
	// Instructions are your team's standing instructions for this customer's orders.
	Instructions                *string                       `audit:"instructions"`
	Email                       *string                       `audit:"email"`
	Phone                       *string                       `audit:"phone"`
	URL                         *string                       `audit:"url"`
	CarrierBillingType          *constants.CarrierBillingType `audit:"carrier_billing_type"`
	CarrierBillingAccount       *string                       `audit:"carrier_billing_account"`
	CreditLimitID               *string                       `audit:"credit_limit_id"`
	CreditLimitValue            *string                       `audit:"credit_limit_value"`
	CreditLimitUnitID           *string                       `audit:"credit_limit_unit_id"`
	CreditLimitUnitAbbreviation *string
	CreditLimitUnitName         *string
	CreditLimitUnitType         *string
	// CreditLimit is the credit limit as money.
	CreditLimit                        *Money  `audit:"-"`
	AcceptsInvoiceEmails               bool    `audit:"accepts_invoice_emails"`
	DefaultCarrierID                   *string `audit:"default_carrier_id"`
	DefaultCarrierName                 *string
	DefaultCarrierIsPortalEnabled      *bool
	DefaultCarrierCreatedAt            *time.Time
	DefaultCarrierUpdatedAt            *time.Time
	DefaultServiceLevelID              *string `audit:"default_service_level_id"`
	DefaultServiceLevelName            *string
	DefaultServiceLevelToken           *string
	DefaultServiceLevelIsPortalEnabled *bool
	DefaultServiceLevelCreatedAt       *time.Time
	DefaultServiceLevelUpdatedAt       *time.Time
	DefaultPaymentTermID               *string `audit:"default_payment_term_id"`
	DefaultPaymentTermName             *string
	DefaultPaymentTermIsActive         *bool
	DefaultPaymentTermCreatedAt        *time.Time
	DefaultPaymentTermUpdatedAt        *time.Time
	DefaultShippingTermID              *string `audit:"default_shipping_term_id"`
	DefaultShippingTermName            *string
	DefaultShippingTermType            *constants.ShippingTermType
	DefaultShippingTermCreatedAt       *time.Time
	DefaultShippingTermUpdatedAt       *time.Time
	DefaultSalesRepID                  *string `audit:"default_sales_rep_id"`
	DefaultSalesRepName                *string
	DefaultSalesRepStatus              *constants.AccountUserStatus
	DefaultSalesRepCreatedAt           *time.Time
	DefaultSalesRepUpdatedAt           *time.Time
	BillToAddressID                    *string `audit:"bill_to_address_id"`
	ShipToAddressID                    *string `audit:"ship_to_address_id"`
	BillToAddress                      *CustomerAddress
	ShipToAddress                      *CustomerAddress
	TypeGroupID                        *string `audit:"type_group_id"`
	TypeGroupName                      *string
	TypeGroupCommissionPolicy          *constants.CommissionPolicy
	TypeGroupFreightPolicy             *constants.FreightPolicy
	TypeGroupType                      *constants.AccountGroupType
	TypeGroupCreatedAt                 *time.Time
	TypeGroupUpdatedAt                 *time.Time
	PriceGroups                        []CustomerAccountGroup `audit:"price_groups"`
	// PriceListIDs are the price lists assigned to the customer directly; set by batch reads.
	PriceListIDs            []string `audit:"-"`
	ParentAccountID         *string  `audit:"parent_account_id"`
	ParentAccountRelationID *string
	ParentAccountName       *string
	ParentAccountNumber     *string
	ParentAccountCreatedAt  *time.Time
	ParentAccountUpdatedAt  *time.Time
	ChildAccounts           []CustomerChildAccount
	// NotificationRecipients are loaded only when asked for.
	NotificationRecipients []NotificationRecipient `audit:"-"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// CustomerChildAccount is the lightweight stub returned when `?include=child_accounts` is requested on a customer resource.
type CustomerChildAccount struct {
	ID        string
	Name      string
	Number    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CustomerAddress represents a customer's address with geolocation.
type CustomerAddress struct {
	ID          string
	Name        string
	Phone       *string
	Email       *string
	IsDropShip  bool
	Geolocation *CustomerGeolocation
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CustomerGeolocation represents a geolocation record.
type CustomerGeolocation struct {
	ID          string
	StreetLine1 *string
	StreetLine2 *string
	Locality    *string
	State       *string
	PostalCode  *string
	Country     string
}

// CustomerAccountGroup is a lightweight account group reference.
type CustomerAccountGroup struct {
	ID               string
	Name             string
	CommissionPolicy constants.CommissionPolicy
	FreightPolicy    constants.FreightPolicy
	Type             constants.AccountGroupType
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CustomerContact is someone who signs in to a customer account.
type CustomerContact struct {
	CustomerAccountID string
	Name              *string
	Email             *string
}

// ListCustomersParams holds the parameters for listing customers.
type ListCustomersParams struct {
	AccountID        string
	Cursor           *string
	Limit            int32
	Query            *string
	CustomerGroupIDs []string
	PriceListIDs     []string
	PricingGroupIDs  []string
	SalesRepIDs      []string
	StatusCodes      []string
	// IsActive narrows to active (true) or archived (false) customers; nil lists both.
	IsActive *bool
	// RelationshipTypes narrows to standalone, parent or child customers.
	RelationshipTypes     []constants.CustomerRelationshipType
	ShippingTermIDs       []string
	PaymentTermIDs        []string
	CommissionPolicyCodes []string
	FreightPolicyCodes    []string
	CarrierIDs            []string
	ServiceLevelIDs       []string
	IsParentAccount       *bool
	City                  *string
	State                 *string
	PostalCode            *string
	StartDate             *time.Time
	EndDate               *time.Time
	Created               CreatedRange
	Includes              []string
}

// ListCustomersResult holds the result of listing customers.
type ListCustomersResult struct {
	Items    []*Customer
	PageInfo pagination.PageInfo
}

// CreateCustomerParams holds the parameters for creating a customer.
type CreateCustomerParams struct {
	OwnerAccountID        string
	Name                  string
	Number                *string
	Instructions          *string
	Email                 *string
	Phone                 *string
	URL                   *string
	StatusCode            *string
	ArchiveStatus         constants.ArchiveStatus
	CommissionPolicy      *constants.CommissionPolicy
	FreightPolicy         *constants.FreightPolicy
	DefaultLeadTimeDays   *int32
	ReceiveCalendarID     *string
	Supply                *constants.Supply
	ParentCustomerID      *string
	ParentRelationID      *string
	DefaultCarrierID      *string
	DefaultServiceLevelID *string
	DefaultPaymentTermID  *string
	DefaultShippingTermID *string
	DefaultSalesRepID     *string
	BillToAddressID       *string
	ShipToAddressID       *string
	BillToAddress         *CreateAddressParams
	ShipToAddress         *CreateAddressParams
	CustomerPriceGroupIDs []string
	// PriceListIDs, when non-nil, are the price lists assigned to the customer directly.
	PriceListIDs          []string
	CustomerTypeGroupID   *string
	CarrierBillingType    *string
	CarrierBillingAccount *string
	CreditLimitValue      *string
	CreditLimitUnitID     *string
	CreditLimitID         *string
	Includes              []string
}

// DeleteCustomerParams holds the parameters for deleting a customer.
type DeleteCustomerParams struct {
	OwnerAccountID    string
	CustomerAccountID string
}

// BulkDeleteCustomersParams holds the parameters for bulk deleting customers.
type BulkDeleteCustomersParams struct {
	OwnerAccountID string
	CustomerIDs    []string
}

// MergeCustomersParams holds the parameters for merging customers.
type MergeCustomersParams struct {
	OwnerAccountID    string
	TargetCustomerID  string
	SourceCustomerIDs []string
	Includes          []string
}

// OrderSalesRep is an order's ID and its current sales rep.
type OrderSalesRep struct {
	OrderID    string
	SalesRepID *string
}

// RelationPriceGroup is a lightweight reference to an account relation price group.
type RelationPriceGroup struct {
	ID             string
	AccountGroupID string
}

// RelationProductLine is a lightweight reference to an account relation product line.
type RelationProductLine struct {
	ID            string
	ProductLineID string
}

// AccountUserRef is a lightweight reference to an account user.
type AccountUserRef struct {
	ID     string
	UserID string
}

// UpdateCustomerParams holds the parameters for updating a customer.
type UpdateCustomerParams struct {
	OwnerAccountID      string
	CustomerAccountID   string
	Name                *string
	Number              *string
	Instructions        field.Clearable[string]
	Email               field.Clearable[string]
	Phone               field.Clearable[string]
	URL                 field.Clearable[string]
	StatusCode          *string
	ArchiveStatus       *constants.ArchiveStatus
	CommissionPolicy    *constants.CommissionPolicy
	FreightPolicy       *constants.FreightPolicy
	DefaultLeadTimeDays field.Clearable[int32]
	ReceiveCalendarID   field.Clearable[string]
	Supply              field.Clearable[constants.Supply]
	ParentCustomerID    field.Clearable[string]
	// ParentRelationID is the parent's relation once ParentCustomerID is resolved.
	ParentRelationID *string
	// NotificationRecipients, when HasNotificationRecipients, replaces the customer's recipients.
	NotificationRecipients    []NotificationRecipientInput
	HasNotificationRecipients bool
	DefaultCarrierID          *string
	DefaultServiceLevelID     field.Clearable[string]
	DefaultPaymentTermID      *string
	DefaultShippingTermID     *string
	DefaultSalesRepID         field.Clearable[string]
	BillToAddressID           field.Clearable[string]
	ShipToAddressID           field.Clearable[string]
	CustomerPriceGroupIDs     []string
	HasCustomerPriceGroupIDs  bool
	// PriceListIDs, when HasPriceListIDs, replaces the price lists assigned to the customer directly.
	PriceListIDs          []string
	HasPriceListIDs       bool
	CustomerTypeGroupID   *string
	CarrierBillingType    *string
	CarrierBillingAccount field.Clearable[string]
	CreditLimit           field.Clearable[fieldpb.QuantityInput]
	CreditLimitID         *string
	Includes              []string
}

// FrequentlyOrderedProduct represents a product frequently ordered by a customer.
type FrequentlyOrderedProduct struct {
	ItemID     string
	OrderCount int32
}

// SyncStripeCustomerEvent is the outbox command payload for reconciling one customer with the account's connected Stripe integration.
//
// It carries identifiers only, never the field values that triggered it: the consumer re-reads the customer at handling time, so a burst of edits collapses into the same final state instead of racing stale snapshots onto Stripe in arbitrary delivery order.
type SyncStripeCustomerEvent struct {
	// OwnerAccountID is the merchant account whose Stripe integration is written to.
	OwnerAccountID string `json:"owner_account_id"`
	// CustomerAccountID is the counterparty account being synced.
	CustomerAccountID string `json:"customer_account_id"`
}

// CustomerLeadTime is the ship-by commitment a new order for one customer would be given, and the rule that produced it.
type CustomerLeadTime struct {
	CustomerAccountID string
	Days              int
	SourceCode        string
	// AccountGroupID is set only when the group is the rule that won.
	AccountGroupID *string
	// ParentCustomerAccountID is set only when the parent account is the rule that won.
	ParentCustomerAccountID *string
}

// CustomerInUse words the 409 for deleting a customer that orders, shipments or child customers still reference.
var CustomerInUse = References{Noun: "customer", Archivable: true}

// MergeCustomersJob is what a customer merge job runs: the sources folded into the target.
type MergeCustomersJob struct {
	TargetCustomerID  string
	SourceCustomerIDs []string
}
