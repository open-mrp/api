package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleProductID = "pd_07oe0r7adh2w"

// A catalog entry as it is sold: an inventory item together with its product type, product line, and customer portal visibility.
//
// Every product is backed by exactly one item, which carries the SKU, description, pricing, attributes, and inventory position. Creating a product creates that item; deleting the product deletes it.
type Product struct {
	// Product ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product"`
	// Product type code, which determines how the product behaves on orders and invoices.
	//
	// - `sale`: a standard sellable product.
	// - `service`: a non-physical service line, such as labor or installation.
	// - `shipping`: a shipping charge applied to an order.
	// - `credit`: a credit applied against an order or invoice.
	// - `return`: a returned product (RMA).
	// - `tax`: a tax line.
	Type constants.ProductTypeCode `json:"type" validate:"required"`
	// Whether the product is shown to buyers in the customer portal.
	//
	// - `visible`: buyers can see and order the product in the portal.
	// - `hidden`: the product is concealed from the portal but remains usable internally.
	//
	// Visibility alone is not enough to expose a product: a buyer only sees it if their account has also been granted access to the product's product line.
	PortalVisibility constants.CustomerPortalVisibility `json:"portal_visibility" validate:"required"`
	// The product line this product is assigned to, if any.
	//
	// Customer accounts are granted access to whole product lines, so this assignment is what decides which buyers can see and order the product. A product with no product line is never visible in the customer portal. The line also supplies the default commission and freight policies for the product.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// The inventory item backing this product, which holds its SKU, description, pricing, and attributes.
	Item *Item `json:"item" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProduct = &Product{
	ID:               SampleProductID,
	Object:           constants.ObjectTypeProduct,
	Type:             SampleProductTypeCode,
	PortalVisibility: constants.CustomerPortalVisibilityVisible,
	ProductLine:      SampleProductLine,
	Item:             SampleItem,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Product) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProduct)
}

// The outcome of a SKU lookup: the products that matched, addressed by the caller's own keys.
type ValidateProductsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=map"`
	// Matched products keyed by the same keys supplied in the request's `products_map`.
	//
	// Keys whose SKU did not match any product are omitted.
	Products map[string]*Product `json:"products" validate:"required"`
}

var SampleValidateProductsResponse = &ValidateProductsResponse{
	Object:   constants.ObjectTypeMap,
	Products: map[string]*Product{"0": SampleProduct},
}

func (*ValidateProductsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleValidateProductsResponse)
}
