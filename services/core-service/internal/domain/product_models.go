package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
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
	ItemID          string // generated in service
	UnitValueRateID string // generated in service
	UnitCostRateID  string // generated in service
	BurnRateRateID  string // generated in service
	SKU             string
	Description     *string
	Notes           *string
	ProductTypeCode string
	ProductLineID   *string
	CategoryID      string
	IsPortalReady   bool
	// UnitPrice / UnitCost / BurnRate are initial rate values written into the
	// unit_value, unit_cost, and burn_rate Rate records. When nil the rate is
	// initialized to "0" against the category's base unit on both sides. When
	// set, UnitPrice and UnitCost additionally enforce the
	// currency-numerator / non-currency-denominator rule (BurnRate does not).
	UnitPrice *CreateRateParams
	UnitCost  *CreateRateParams
	BurnRate  *CreateRateParams
	// AttributeIDs are connected to the new item at creation time.
	AttributeIDs []string
	Includes     []string
}

// UpdateProductParams holds parameters for partially updating a product.
type UpdateProductParams struct {
	AccountID         string
	ProductID         string
	SKU               *string
	Description       *string
	UpdateDescription bool
	Notes             *string
	UpdateNotes       bool
	IsPortalReady     *bool
	UnitPrice         *CreateRateParams
	Includes          []string
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
}

// ValidateProductsResult contains matched products keyed by the original map key.
type ValidateProductsResult struct {
	Products map[string]*ProductFull
}
