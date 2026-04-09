package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

// Customer represents a full customer record from the database.
type Customer struct {
	ID                            string
	Name                          string                        `audit:"name"`
	Number                        string                        `audit:"number"`
	Status                        constants.AccountStatusCode   `audit:"status"`
	IsEdiEnabled                  bool                          `audit:"is_edi_enabled"`
	IsParentAccount               bool                          `audit:"is_parent_account"`
	CommissionPolicy              constants.CommissionPolicy    `audit:"commission_policy"`
	FreightPolicy                 constants.FreightPolicy       `audit:"freight_policy"`
	Note                          *string                       `audit:"note"`
	Email                         *string                       `audit:"email"`
	Phone                         *string                       `audit:"phone"`
	URL                           *string                       `audit:"url"`
	CarrierBillingType            *constants.CarrierBillingType `audit:"carrier_billing_type"`
	CarrierBillingAccount         *string                       `audit:"carrier_billing_account"`
	AcceptsInvoiceEmails          bool                          `audit:"accepts_invoice_emails"`
	DefaultCarrierID              *string                       `audit:"default_carrier_id"`
	DefaultCarrierName            *string
	DefaultCarrierIsPortalEnabled *bool
	DefaultServiceLevelID         *string `audit:"default_service_level_id"`
	DefaultServiceLevelName       *string
	DefaultPaymentTermID          *string `audit:"default_payment_term_id"`
	DefaultPaymentTermName        *string
	DefaultPaymentTermIsActive    *bool
	DefaultShippingTermID         *string `audit:"default_shipping_term_id"`
	DefaultShippingTermName       *string
	DefaultShippingTermType       *constants.ShippingTermType
	DefaultPriorityID             *string
	DefaultPriorityCode           *constants.PriorityCode `audit:"default_priority_code"`
	DefaultPriorityName           *string
	DefaultSalesRepID             *string `audit:"default_sales_rep_id"`
	DefaultSalesRepName           *string
	BillToAddressID               *string          `audit:"bill_to_address_id"`
	ShipToAddressID               *string          `audit:"ship_to_address_id"`
	BillToAddress                 *CustomerAddress
	ShipToAddress                 *CustomerAddress
	TypeGroupID                   *string `audit:"type_group_id"`
	TypeGroupName                 *string
	TypeGroupCommissionPolicy     *constants.CommissionPolicy
	TypeGroupFreightPolicy        *constants.FreightPolicy
	TypeGroupType                 *constants.AccountGroupType
	PriceGroups                   []CustomerAccountGroup `audit:"price_groups"`
	ParentAccountID               *string                `audit:"parent_account_id"`
	ParentAccountName             *string
	ParentAccountNumber           *string
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
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
	ID   string
	Name string
}

// CustomerSummary is a lightweight customer record for list results.
type CustomerSummary struct {
	ID                string
	Name              string
	Number            string
	Email             *string
	CustomerTypeGroup *string
	Status            constants.AccountStatusCode
	CreatedAt         time.Time
}

// ListCustomersParams holds the parameters for listing customers.
type ListCustomersParams struct {
	AccountID             string
	Cursor                *string
	Limit                 int32
	Query                 *string
	CustomerGroupIDs      []string
	PricingGroupIDs       []string
	SalesRepIDs           []string
	StatusCodes           []string
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
}

// ListCustomersResult holds the result of listing customers.
type ListCustomersResult struct {
	Items    []*CustomerSummary
	PageInfo pagination.PageInfo
}

// CreateCustomerParams holds the parameters for creating a customer.
type CreateCustomerParams struct {
	OwnerAccountID        string
	Name                  string
	Number                *string
	Note                  *string
	Email                 *string
	Phone                 *string
	URL                   *string
	StatusCode            *string
	IsEdiEnabled          *bool
	CommissionPolicy      *constants.CommissionPolicy
	FreightPolicy         *constants.FreightPolicy
	DefaultCarrierID      *string
	DefaultServiceLevelID *string
	DefaultPaymentTermID  *string
	DefaultShippingTermID *string
	DefaultPriorityCode   *string
	DefaultSalesRepUserID *string
	BillToAddressID       *string
	ShipToAddressID       *string
	BillToAddress         *CreateAddressParams
	ShipToAddress         *CreateAddressParams
	CustomerPriceGroupIDs []string
	CustomerTypeGroupID   *string
	CarrierBillingType    *string
	CarrierBillingAccount *string
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
	OwnerAccountID           string
	CustomerAccountID        string
	Name                     *string
	Number                   *string
	Note                     *string
	Email                    *string
	Phone                    *string
	URL                      *string
	StatusCode               *string
	IsEdiEnabled             *bool
	CommissionPolicy         *constants.CommissionPolicy
	FreightPolicy            *constants.FreightPolicy
	DefaultCarrierID         *string
	DefaultServiceLevelID    *string
	DefaultPaymentTermID     *string
	DefaultShippingTermID    *string
	DefaultPriorityCode      *string
	DefaultSalesRepUserID    *string
	BillToAddressID          *string
	ShipToAddressID          *string
	CustomerPriceGroupIDs    []string
	HasCustomerPriceGroupIDs bool
	CustomerTypeGroupID      *string
	CarrierBillingType       *string
	CarrierBillingAccount    *string
}

// FrequentlyOrderedProduct represents a product frequently ordered by a customer.
type FrequentlyOrderedProduct struct {
	ItemID           string
	ProductName      string
	UnitID           *string
	UnitAbbreviation *string
	OrderCount       int32
}
