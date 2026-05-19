package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Product line available in the catalog.
type CatalogProductLine struct {
	// Product line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_product_line"`
	// Name.
	Name string `json:"name" validate:"required"`
}

// Category of products in the catalog.
type CatalogCategory struct {
	// Item category ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_category"`
	// Name.
	Name string `json:"name" validate:"required"`
	// Properties associated with this item category.
	Properties *List[CatalogProperty] `json:"properties" validate:"required"`
	// Products in this category.
	Products *List[CatalogProduct] `json:"products" validate:"required"`
}

// Property associated with an item category.
type CatalogProperty struct {
	// Property ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_property"`
	// Name.
	Name string `json:"name" validate:"required"`
}

// Product in the catalog.
type CatalogProduct struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_product"`
	// Associated item.
	Item *Item `json:"item" validate:"required"`
	// Description.
	Description string `json:"description" validate:"required"`
	// Attributes.
	Attributes *List[CatalogAttribute] `json:"attributes" validate:"required"`
}

// Attribute of a product in the catalog.
type CatalogAttribute struct {
	// Attribute ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_attribute"`
	// Attribute value.
	Name string `json:"name" validate:"required"`
	// Property this attribute belongs to.
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
	Object: constants.ObjectTypeCatalogProduct,
	Item: &Item{
		ID:     SampleItemID,
		Object: constants.ObjectTypeItem,
		SKU:    SampleItemSKU,
	},
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
