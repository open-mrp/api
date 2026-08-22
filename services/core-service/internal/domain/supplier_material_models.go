package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
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
