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
)

// CreateProductRequest is the request to create a product.
type CreateProductRequest struct {
	// SKU.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description.
	Description *string `json:"description"`
	// Notes.
	Notes *string `json:"notes"`
	// Product type code (e.g. sale, sample).
	ProductTypeCode constants.ProductTypeCode `json:"type" validate:"required"`
	// Product line ID.
	ProductLineID *string `json:"product_line_id" validate:"omitempty"`
	// Category ID.
	CategoryID string `json:"category_id" validate:"required"`
	// Whether visible in the customer portal.
	PortalVisibility *constants.CustomerPortalVisibility `json:"portal_visibility,omitempty" default:"hidden" nullable:"false"`
	// Initial unit price. When set, numerator must be a currency unit and
	// denominator must not be.
	UnitPrice *apirequest.RateInput `json:"unit_price,omitempty"`
	// Initial unit cost. Same currency rule as unit_price.
	UnitCost *apirequest.RateInput `json:"unit_cost,omitempty"`
	// Initial burn rate (waste / scrap). No currency requirement.
	BurnRate *apirequest.RateInput `json:"burn_rate,omitempty"`
	// Attribute IDs to connect to the product at creation time.
	AttributeIDs []string `json:"attribute_ids,omitempty"`
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
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
