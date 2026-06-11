package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// CreateProductRequest is the request to create a product.
type CreateProductRequest struct {
	// Stock keeping unit code for the product's item.
	//
	// Must be unique within the account; creation fails with a conflict error if another item already uses it.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description.
	Description field.Optional[string] `json:"description,omitzero"`
	// Notes.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Product type code, which determines how the product behaves on orders and invoices.
	//
	// - `sale`: a standard sellable product.
	// - `service`: a non-physical service line, such as labor or installation.
	// - `shipping`: a shipping charge applied to an order.
	// - `credit`: a credit applied against an order or invoice.
	// - `return`: a returned product (RMA).
	// - `tax`: a tax line.
	ProductTypeCode constants.ProductTypeCode `json:"type" validate:"required"`
	// ID of the product line to assign the product to.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero" validate:"omitempty"`
	// ID of the item category for the product's item.
	//
	// The category's unit group determines the default units used for the product's pricing rates and inventory tracking.
	CategoryID string `json:"category_id" validate:"required"`
	// Whether the product is shown to buyers in the customer portal.
	//
	// - `visible`: buyers can see and order the product in the portal.
	// - `hidden`: the product is concealed from the portal but remains usable internally.
	//
	// When omitted, the product is created hidden, so it must be set to `visible` before buyers can see it.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero" default:"hidden"`
	// Initial selling price per unit.
	//
	// When set, the numerator unit must be a currency unit and the denominator unit must not be. When omitted, the price is initialized to a zero rate in the category's base unit.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Initial cost per unit.
	//
	// The same unit rule as `unit_price` applies: the numerator unit must be a currency unit and the denominator unit must not be. When omitted, the cost is initialized to a zero rate in the category's base unit.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// Attribute IDs to connect to the product at creation time.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
}

var sampleCreateProductRequest = &CreateProductRequest{
	SKU:             apiresource.SampleItemSKU,
	ProductTypeCode: apiresource.SampleProductTypeCode,
	CategoryID:      apiresource.SampleItemCategoryID,
}

func (*CreateProductRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductRequest)
}

// Creates a product and its backing inventory item.
//
// The new item starts with zero on-hand inventory, and its pricing defaults to zero rates in the category's base unit unless `unit_price` or `unit_cost` is provided.
type CreateProductEndpoint struct{}

func (e *CreateProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*CreateProductRequest, *apiresource.Product]{
		Title:             "Create Product",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/products",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).CreateProduct
		},
		LocationFunc: func(resp *apiresource.Product) string {
			return "/v1/catalog/products/" + resp.ID
		},
		ObjectType: constants.ObjectTypeProduct,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
