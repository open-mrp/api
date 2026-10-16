package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

// SupplierMaterial represents a link between a supplier and a material.
type SupplierMaterial struct {
	ID                  string
	MaterialID          string
	SupplierAccountID   string
	SupplierPartNumber  string  `audit:"supplier_part_number"`
	SupplierDescription *string `audit:"supplier_description"`
	IsActive            bool    `audit:"is_active"`
	OwnerAccountID      string
	CreatedAt           time.Time
	UpdatedAt           time.Time

	// Joined
	Material *Material
}

type ListSupplierMaterialsParams struct {
	SupplierAccountID string
	OwnerAccountID    string
	Cursor            *string
	Limit             int32
	Query             *string
}

type ListSupplierMaterialsResult struct {
	SupplierMaterials []*SupplierMaterial
	PageInfo          pagination.PageInfo
}

type CreateSupplierMaterialParams struct {
	OwnerAccountID      string
	MaterialID          string
	SupplierAccountID   string
	SupplierPartNumber  string
	SupplierDescription *string
	IsActive            bool
}

type UpdateSupplierMaterialParams struct {
	OwnerAccountID      string
	SupplierAccountID   string
	MaterialID          string
	SupplierPartNumber  *string
	SupplierDescription *string
	UpdateDescription   bool
	IsActive            *bool
}

type DeleteSupplierMaterialParams struct {
	OwnerAccountID    string
	SupplierAccountID string
	MaterialID        string
}

// SupplierItem is an item a supplier provides, with the supplier's own code and description for it. It is the
// forge.1 view of a supplier material, addressed by its own ID rather than its material's.
type SupplierItem struct {
	ID                  string
	SupplierAccountID   string
	ItemID              string
	MaterialID          string
	SupplierSKU         string                  `audit:"supplier_sku"`
	SupplierDescription *string                 `audit:"supplier_description"`
	Status              constants.ArchiveStatus `audit:"status"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ListSupplierItemsParams holds the parameters for listing a supplier's items.
type ListSupplierItemsParams struct {
	OwnerAccountID    string
	SupplierAccountID string
	Cursor            *string
	Limit             int32
	Query             *string
	// IsActive narrows to active (true) or archived (false) supplier items; nil lists both.
	IsActive *bool
	ItemIDs  []string
	Created  CreatedRange
}

// ListSupplierItemsResult holds a page of supplier items.
type ListSupplierItemsResult struct {
	SupplierItems []*SupplierItem
	PageInfo      pagination.PageInfo
}

// CreateSupplierItemParams holds the parameters for creating a supplier item.
type CreateSupplierItemParams struct {
	OwnerAccountID      string
	SupplierAccountID   string
	ItemID              string
	SupplierSKU         string
	SupplierDescription *string
	ArchiveStatus       constants.ArchiveStatus
}

// UpdateSupplierItemParams holds the parameters for updating a supplier item. Only the fields sent change.
type UpdateSupplierItemParams struct {
	OwnerAccountID      string
	SupplierAccountID   string
	SupplierItemID      string
	SupplierSKU         *string
	SupplierDescription field.Clearable[string]
	ArchiveStatus       *constants.ArchiveStatus
}
