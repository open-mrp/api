package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// Supplier represents a full supplier record.
type Supplier struct {
	ID            string
	Name          string           `audit:"name"`
	Number        string           `audit:"number"`
	Note          *string          `audit:"note"`
	BillToAddress *CustomerAddress `audit:"bill_to_address"`
	ShipToAddress *CustomerAddress `audit:"ship_to_address"`
	MaterialCount int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SupplierSummary is a lightweight supplier record for list results.
//
// The two default addresses are always carried as ids, and as whole records when the caller asked
// for them. They cannot be resolved downstream: a supplier's addresses belong to the supplier's own
// account, which the gateway's account-scoped address loader cannot read.
type SupplierSummary struct {
	ID              string
	Name            string
	Number          string
	Note            *string
	BillToAddressID *string
	ShipToAddressID *string
	BillToAddress   *CustomerAddress
	ShipToAddress   *CustomerAddress
	MaterialCount   int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ListSuppliersParams holds the parameters for listing suppliers.
type ListSuppliersParams struct {
	OwnerAccountID string
	Cursor         *string
	Limit          int32
	Query          *string
	ItemIDs        []string
	StartDate      *time.Time
	EndDate        *time.Time
	Includes       []string
}

// GetSupplierParams holds the parameters for retrieving a single supplier.
type GetSupplierParams struct {
	OwnerAccountID string
	SupplierID     string
	Includes       []string
}

// ListSuppliersResult holds the result of listing suppliers.
type ListSuppliersResult struct {
	Items    []*SupplierSummary
	PageInfo pagination.PageInfo
}

// CreateSupplierParams holds the parameters for creating a supplier.
type CreateSupplierParams struct {
	OwnerAccountID string
	Name           string
	Number         string
	Note           *string
	BillToAddress  *CreateAddressParams
	ShipToAddress  *CreateAddressParams
	Includes       []string
}

// UpdateSupplierParams holds the parameters for updating a supplier.
type UpdateSupplierParams struct {
	OwnerAccountID  string
	SupplierID      string
	Name            *string
	Number          *string
	Note            *string
	UpdateNote      bool
	BillToAddressID *string
	ShipToAddressID *string
	Includes        []string
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
