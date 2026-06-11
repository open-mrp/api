package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductID = "pd_013c29ab3f1518d0004094c316"

// Product with expandable item, product line, and product type.
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
	PortalVisibility constants.CustomerPortalVisibility `json:"portal_visibility" validate:"required"`
	// Product line.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// Item.
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

// ValidateProductsResponse is the response for the validate products endpoint.
type ValidateProductsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=map"`
	// Validated products keyed by original map key.
	Products map[string]*Product `json:"products" validate:"required"`
}

var SampleValidateProductsResponse = &ValidateProductsResponse{
	Object:   constants.ObjectTypeMap,
	Products: map[string]*Product{"0": SampleProduct},
}

func (*ValidateProductsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleValidateProductsResponse)
}
