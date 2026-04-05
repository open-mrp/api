package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// CatalogProductLine represents a product line available in the catalog.
type CatalogProductLine struct {
	// The unique identifier for the product line.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_product_line"`
	// The name of the product line.
	Name string `json:"name" validate:"required"`
}

// CatalogCategory represents a category of products in the catalog.
type CatalogCategory struct {
	// The unique identifier for the item category.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_category"`
	// The name of the item category.
	Name string `json:"name" validate:"required"`
	// The properties associated with this item category.
	Properties []CatalogProperty `json:"properties" validate:"required"`
	// The products in this category.
	Products []CatalogProduct `json:"products" validate:"required"`
}

// CatalogProperty represents a property associated with an item category.
type CatalogProperty struct {
	// The unique identifier for the property.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_property"`
	// The name of the property.
	Name string `json:"name" validate:"required"`
}

// CatalogProduct represents a product in the catalog.
type CatalogProduct struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_product"`
	// The item associated with this catalog product.
	Item *Item `json:"item" validate:"required"`
	// The product description.
	Description string `json:"description" validate:"required"`
	// The attributes of this product.
	Attributes []CatalogAttribute `json:"attributes" validate:"required"`
}

// CatalogAttribute represents an attribute of a product in the catalog.
type CatalogAttribute struct {
	// The unique identifier for the attribute.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=catalog_attribute"`
	// The attribute value (text).
	Name string `json:"name" validate:"required"`
	// The property this attribute belongs to.
	Property *CatalogProperty `json:"property" validate:"required"`
}

var SampleCatalogProductLine = &CatalogProductLine{
	ID:     "pl_01jm4r6700e3kxb9w2nqh7g5fp",
	Object: constants.ObjectTypeCatalogProductLine,
	Name:   "Industrial Fasteners",
}

var SampleCatalogAttribute = &CatalogAttribute{
	ID:     "at_01jm4r6700e3kxb9w2nqh7g5fp",
	Object: constants.ObjectTypeCatalogAttribute,
	Name:   "Red",
	Property: &CatalogProperty{
		ID:     "pr_01jm4r6700e3kxb9w2nqh7g5fp",
		Object: constants.ObjectTypeCatalogProperty,
		Name:   "Color",
	},
}

var SampleCatalogProduct = &CatalogProduct{
	Object: constants.ObjectTypeCatalogProduct,
	Item: &Item{
		ID:     "it_01jm4r6700e3kxb9w2nqh7g5fp",
		Object: constants.ObjectTypeItem,
		SKU:    "WDG-001",
	},
	Description: "Hex Bolt M10x30 Zinc",
	Attributes:  []CatalogAttribute{*SampleCatalogAttribute},
}

var SampleCatalogProperty = &CatalogProperty{
	ID:     "pr_01jm4r6700e3kxb9w2nqh7g5fp",
	Object: constants.ObjectTypeCatalogProperty,
	Name:   "Color",
}

var SampleCatalogCategory = &CatalogCategory{
	ID:         "ic_01jm4r6700e3kxb9w2nqh7g5fp",
	Object:     constants.ObjectTypeCatalogCategory,
	Name:       "Finished Goods",
	Properties: []CatalogProperty{*SampleCatalogProperty},
	Products:   []CatalogProduct{*SampleCatalogProduct},
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
