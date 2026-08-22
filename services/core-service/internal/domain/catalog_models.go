package domain

import "github.com/open-mrp/api/shared/pagination"

// CatalogProductLine represents a product line available in the catalog.
type CatalogProductLine struct {
	ID   string
	Name string
}

// ListCatalogProductLinesParams contains parameters for listing catalog product lines.
type ListCatalogProductLinesParams struct {
	Cursor *string
	Limit  int32
	Query  *string
}

// ListCatalogProductLinesResult contains the result of listing catalog product lines.
type ListCatalogProductLinesResult struct {
	ProductLines []*CatalogProductLine
	PageInfo     pagination.PageInfo
}

// CatalogCategory represents a category of products in the catalog, grouped by item category.
type CatalogCategory struct {
	ID         string
	Name       string
	Properties []*CatalogProperty
	Products   []*CatalogProduct
}

// ListCatalogProductsParams contains parameters for listing catalog products.
type ListCatalogProductsParams struct {
	ProductLineID string
	Cursor        *string
	Limit         int32
	Query         *string
}

// ListCatalogProductsResult contains the result of listing catalog products.
type ListCatalogProductsResult struct {
	Categories []*CatalogCategory
	PageInfo   pagination.PageInfo
}

// CatalogProperty represents a property associated with an item category.
type CatalogProperty struct {
	ID   string
	Name string
}

// CatalogProduct represents a product in the catalog.
type CatalogProduct struct {
	ItemID      string
	SKU         string
	Description string
	Attributes  []*CatalogAttribute
}

// CatalogAttribute represents an attribute of a product in the catalog.
type CatalogAttribute struct {
	ID           string
	Name         string
	PropertyID   string
	PropertyName string
}
