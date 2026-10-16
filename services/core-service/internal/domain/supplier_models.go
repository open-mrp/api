package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

// SupplierIncludeAddresses and SupplierIncludeMaterialCount are the includes a supplier read accepts. The default
// addresses belong to the supplier's own account, which the gateway's account-scoped address loader cannot read, so
// they are joined here when asked for.
const (
	SupplierIncludeBillToAddress = "bill_to_address"
	SupplierIncludeShipToAddress = "ship_to_address"
	SupplierIncludeMaterialCount = "material_count"
)

// Supplier is an account the owner buys from, read through the owner's relation to it.
type Supplier struct {
	ID     string
	Name   string `audit:"name"`
	Number string `audit:"number"`
	// ArchiveStatus is whether the supplier can be named on new records.
	ArchiveStatus constants.ArchiveStatus `audit:"status"`
	// Instructions are your team's standing instructions for this supplier's orders.
	Instructions          *string                       `audit:"instructions"`
	Email                 *string                       `audit:"email"`
	Phone                 *string                       `audit:"phone"`
	URL                   *string                       `audit:"url"`
	CarrierBillingType    *constants.CarrierBillingType `audit:"carrier_billing_type"`
	CarrierBillingAccount *string                       `audit:"carrier_billing_account"`
	DefaultCarrierID      *string                       `audit:"default_carrier_id"`
	DefaultServiceLevelID *string                       `audit:"default_service_level_id"`
	PaymentTermID         *string                       `audit:"default_payment_term_id"`
	ShippingTermID        *string                       `audit:"default_shipping_term_id"`
	BillToAddressID       *string                       `audit:"bill_to_address_id"`
	ShipToAddressID       *string                       `audit:"ship_to_address_id"`
	// BillToAddress and ShipToAddress are joined only when asked for.
	BillToAddress *CustomerAddress `audit:"-"`
	ShipToAddress *CustomerAddress `audit:"-"`
	// MaterialCount is how many supplier items the supplier has, counted only when asked for; derived, so not audited.
	MaterialCount *int64 `audit:"-"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SupplierInUse words the 409 for deleting a supplier that purchase orders or supplier items still reference.
var SupplierInUse = References{Noun: "supplier", Archivable: true}

// ListSuppliersParams holds the parameters for listing suppliers.
type ListSuppliersParams struct {
	OwnerAccountID string
	Cursor         *string
	Limit          int32
	Query          *string
	ItemIDs        []string
	// IsActive narrows to active (true) or archived (false) suppliers; nil lists both.
	IsActive *bool
	Created  CreatedRange
	Includes []string
}

// GetSupplierParams holds the parameters for retrieving a single supplier.
type GetSupplierParams struct {
	OwnerAccountID string
	SupplierID     string
	Includes       []string
}

// ListSuppliersResult holds the result of listing suppliers.
type ListSuppliersResult struct {
	Items    []*Supplier
	PageInfo pagination.PageInfo
}

// CreateSupplierParams holds the parameters for creating a supplier.
type CreateSupplierParams struct {
	OwnerAccountID        string
	Name                  string
	Number                string
	ArchiveStatus         constants.ArchiveStatus
	Instructions          *string
	Email                 *string
	Phone                 *string
	URL                   *string
	CarrierBillingType    *constants.CarrierBillingType
	CarrierBillingAccount *string
	DefaultCarrierID      *string
	DefaultServiceLevelID *string
	PaymentTermID         *string
	ShippingTermID        *string
	BillToAddress         *CreateAddressParams
	ShipToAddress         *CreateAddressParams
	Includes              []string
}

// UpdateSupplierParams holds the parameters for updating a supplier. Only the fields sent change.
type UpdateSupplierParams struct {
	OwnerAccountID        string
	SupplierID            string
	Name                  *string
	Number                *string
	ArchiveStatus         *constants.ArchiveStatus
	Instructions          field.Clearable[string]
	Email                 field.Clearable[string]
	Phone                 field.Clearable[string]
	URL                   field.Clearable[string]
	CarrierBillingType    *constants.CarrierBillingType
	CarrierBillingAccount field.Clearable[string]
	DefaultCarrierID      field.Clearable[string]
	DefaultServiceLevelID field.Clearable[string]
	PaymentTermID         field.Clearable[string]
	ShippingTermID        field.Clearable[string]
	BillToAddressID       field.Clearable[string]
	ShipToAddressID       field.Clearable[string]
	Includes              []string
}

// SupplierRelationWrite is every value an update stores on the owner's relation to a supplier, already merged with
// what the relation held.
type SupplierRelationWrite struct {
	OwnerAccountID        string
	SupplierID            string
	Name                  string
	Number                string
	IsActive              bool
	Instructions          *string
	CarrierBillingType    *constants.CarrierBillingType
	CarrierBillingAccount *string
	DefaultCarrierID      *string
	DefaultServiceLevelID *string
	PaymentTermID         *string
	ShippingTermID        *string
	BillToAddressID       *string
	ShipToAddressID       *string
}

// DeleteSupplierParams holds the parameters for deleting a supplier.
type DeleteSupplierParams struct {
	OwnerAccountID string
	SupplierID     string
}

// BulkDeleteSuppliersParams holds the parameters for bulk deleting suppliers.
type BulkDeleteSuppliersParams struct {
	OwnerAccountID string
	SupplierIDs    []string
}
