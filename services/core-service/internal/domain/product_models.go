package domain

import (
	"time"

	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

// ProductFull represents a product entity, which extends an Item with product-specific fields.
type ProductFull struct {
	ID              string
	ProductTypeCode string  `audit:"product_type_code"`
	IsPortalReady   bool    `audit:"is_portal_ready"`
	ProductLineID   *string `audit:"product_line_id"`
	ItemID          string
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// Joined data
	Item        *Item
	ProductLine *ProductLineFull
	ProductType *ProductType
}

// ListProductsFullParams holds parameters for listing products with pagination and filtering.
type ListProductsFullParams struct {
	AccountID      string
	Cursor         *string
	Limit          int32
	Query          *string
	CustomerIDs    []string
	ProductLineIDs []string
	CategoryIDs    []string
	AttributeIDs   []string
	StartDate      *time.Time
	EndDate        *time.Time
	IsPortalReady  *bool
	Includes       []string
}

// ListProductsFullResult contains a page of products plus pagination info.
type ListProductsFullResult struct {
	Products []*ProductFull
	PageInfo pagination.PageInfo
}

// ExportProductsParams holds filter parameters for a full (unpaginated) product export.
type ExportProductsParams struct {
	AccountID      string
	Query          *string
	CustomerIDs    []string
	ProductLineIDs []string
	CategoryIDs    []string
	AttributeIDs   []string
	StartDate      *time.Time
	EndDate        *time.Time
	IsPortalReady  *bool
}

// GetProductFullParams holds parameters for retrieving a single product.
type GetProductFullParams struct {
	AccountID string
	ProductID string
	Includes  []string
}

// CreateProductParams holds parameters for creating a new product.
type CreateProductParams struct {
	AccountID       string
	SKU             string
	Description     *string
	Notes           *string
	ProductTypeCode string
	ProductLineID   *string
	CategoryID      string
	IsPortalReady   bool
	// UnitPrice / UnitCost are initial rate values written into the unit_value and unit_cost Rate records. When nil they default to "0" against the category's base unit on both sides. When set, both enforce the currency-numerator / non-currency-denominator rule. Burn rate is always initialized to "0" per day and recomputed from inventory history.
	UnitPrice *CreateRateParams
	UnitCost  *CreateRateParams
	// AttributeIDs are connected to the new item at creation time.
	AttributeIDs []string
	Includes     []string
}

// InsertProductItemParams is the input for writing a product's item row. It carries
// the service-generated IDs (item id + the three rate ids) alongside the item's
// caller-provided fields, so the generated IDs stay out of CreateProductParams.
type InsertProductItemParams struct {
	ItemID          string
	AccountID       string
	SKU             string
	Description     *string
	Notes           *string
	CategoryID      string
	UnitValueRateID string
	UnitCostRateID  string
	BurnRateRateID  string
}

// UpdateProductParams holds parameters for partially updating a product.
type UpdateProductParams struct {
	AccountID     string
	ProductID     string
	SKU           *string
	Description   field.Clearable[string]
	Notes         field.Clearable[string]
	IsPortalReady *bool
	UnitPrice     *CreateRateParams
	Includes      []string
}

// BulkUpsertProductsParams holds parameters for bulk upserting products, matched by SKU.
type BulkUpsertProductsParams struct {
	Products []UpsertProductParams
}

// UpsertProductParams is a single product to create or update in a bulk upsert. On
// create all fields apply; on update sku/description/notes/portal/unit_price/unit_cost
// and properties are applied (type, product line, and category are create-only, matching
// the single update endpoint). Properties are additive.
type UpsertProductParams struct {
	SKU             string
	ProductTypeCode string // create-only; defaults to "sale" when empty
	Description     *string
	Notes           *string
	Category        ObjectIdentifier  // create-only
	ProductLine     *ObjectIdentifier // create-only; optional
	IsPortalReady   *bool
	UnitPrice       *CreateRateParams
	UnitCost        *CreateRateParams
	// Properties are resolved to attributes (find-or-create by name + value) and attached.
	Properties []UpsertItemPropertyParams
}

// ProductSKUMatch is an existing product keyed by SKU, with the IDs needed to update it.
type ProductSKUMatch struct {
	ProductID       string
	ItemID          string
	SKU             string
	CategoryID      string
	UnitValueRateID string
	UnitCostRateID  string
}

// mirrors an upsert row with its category and product line resolved to ids. Property
// names/values stay as written: they are found-or-created when the job runs.
type ResolvedUpsertProductRow struct {
	SKU             string
	ProductTypeCode string
	Description     *string
	Notes           *string
	CategoryID      string
	ProductLineID   *string
	IsPortalReady   *bool
	UnitPrice       *CreateRateParams
	UnitCost        *CreateRateParams
	Properties      []UpsertItemPropertyParams
}

// DeleteProductParams holds parameters for soft-deleting a product.
type DeleteProductParams struct {
	AccountID string
	ProductID string
}

// ChangeProductProductLineParams holds parameters for changing a product's product line.
type ChangeProductProductLineParams struct {
	AccountID     string
	ProductID     string
	ProductLineID string
	Includes      []string
}

// ValidateProductsParams holds parameters for validating products by SKU.
type ValidateProductsParams struct {
	AccountID   string
	ProductsMap map[string]string // key -> SKU
	Includes    []string
}

// ValidateProductsResult contains matched products keyed by the original map key.
type ValidateProductsResult struct {
	Products map[string]*ProductFull
}
