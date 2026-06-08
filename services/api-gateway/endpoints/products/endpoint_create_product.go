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
	// SKU.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description.
	Description field.Optional[string] `json:"description,omitzero"`
	// Notes.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Product type code (e.g. sale, sample).
	ProductTypeCode constants.ProductTypeCode `json:"type" validate:"required"`
	// Product line ID.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero" validate:"omitempty"`
	// Category ID.
	CategoryID string `json:"category_id" validate:"required"`
	// Whether visible in the customer portal.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero" default:"hidden"`
	// Initial unit price. When set, numerator must be a currency unit and
	// denominator must not be.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Initial unit cost. Same currency rule as unit_price.
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

// Creates a product.
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
