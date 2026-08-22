package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// Product line available in the catalog.
//
// A product line is the top-level grouping of the catalog; browse its products by passing this product line's ID to the list-catalog-products endpoint.
type CatalogProductLine struct {
	// Product line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_product_line"`
	// Display name of the product line.
	Name string `json:"name" validate:"required"`
}

// Category of products in the catalog.
type CatalogCategory struct {
	// Item category ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_category"`
	// Display name of the category.
	Name string `json:"name" validate:"required"`
	// Properties shared by products in this category, such as `Color` or `Size`.
	//
	// These are the dimensions along which the category's products vary; each product's specific values appear under its `attributes`.
	Properties *List[CatalogProperty] `json:"properties" validate:"required"`
	// Products belonging to this category.
	//
	// Every product the category contributes to the requested product line is returned here — pagination applies to categories, not to the products inside them.
	Products *List[CatalogProduct] `json:"products" validate:"required"`
}

// Property associated with an item category, e.g. `Color`.
//
// A property defines a dimension along which products in a category vary; its possible values are represented as catalog attributes.
type CatalogProperty struct {
	// Property ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_property"`
	// Display name of the property, e.g. `Color`.
	Name string `json:"name" validate:"required"`
}

// Product in the catalog.
//
// A catalog product is identified by its underlying `item` rather than a product ID of its own.
type CatalogProduct struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_product"`
	// Inventory item this catalog product represents.
	//
	// Use its `id` and `sku` to look up the full item.
	Item *Item `json:"item" validate:"required"`
	// Human-readable description of the product, carried over from the item.
	Description string `json:"description" validate:"required"`
	// Attribute values that distinguish this product within its category, e.g. `Red` for the `Color` property.
	Attributes *List[CatalogAttribute] `json:"attributes" validate:"required"`
}

// Attribute of a product in the catalog: a single value of a property, e.g. `Red` for the `Color` property.
type CatalogAttribute struct {
	// Attribute ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_attribute"`
	// The attribute's value, e.g. `Red`.
	//
	// This is the specific value the product takes for its `property`.
	Name string `json:"name" validate:"required"`
	// Property this attribute is a value of, e.g. `Color`.
	Property *CatalogProperty `json:"property" validate:"required"`
}

var SampleCatalogProductLine = &CatalogProductLine{
	ID:     SampleProductLineID,
	Object: constants.ObjectTypeCatalogProductLine,
	Name:   "Industrial Fasteners",
}

var SampleCatalogAttribute = &CatalogAttribute{
	ID:     SampleAttributeID,
	Object: constants.ObjectTypeCatalogAttribute,
	Name:   "Red",
	Property: &CatalogProperty{
		ID:     SamplePropertyID,
		Object: constants.ObjectTypeCatalogProperty,
		Name:   "Color",
	},
}

var SampleCatalogProduct = &CatalogProduct{
	Object:      constants.ObjectTypeCatalogProduct,
	Item:        SampleItem,
	Description: "Hex Bolt M10x30 Zinc",
	Attributes:  NewList([]CatalogAttribute{*SampleCatalogAttribute}, PageInfo{}),
}

var SampleCatalogProperty = &CatalogProperty{
	ID:     SamplePropertyID,
	Object: constants.ObjectTypeCatalogProperty,
	Name:   "Color",
}

var SampleCatalogCategory = &CatalogCategory{
	ID:         SampleItemCategoryID,
	Object:     constants.ObjectTypeCatalogCategory,
	Name:       "Finished Goods",
	Properties: NewList([]CatalogProperty{*SampleCatalogProperty}, PageInfo{}),
	Products:   NewList([]CatalogProduct{*SampleCatalogProduct}, PageInfo{}),
}

func (*CatalogProductLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCatalogProductLine)
}

func (*CatalogProduct) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCatalogProduct)
}

func (*CatalogCategory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCatalogCategory)
}

func (*CatalogProperty) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCatalogProperty)
}

func (*CatalogAttribute) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCatalogAttribute)
}
