package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ValidateProductsRequest is the request to validate products by SKU.
type ValidateProductsRequest struct {
	// Map of arbitrary keys to SKU values.
	ProductsMap map[string]string `json:"products_map" validate:"required"`
}

var sampleValidateProductsRequest = &ValidateProductsRequest{
	ProductsMap: map[string]string{"0": apiresource.SampleItemSKU},
}

func (*ValidateProductsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleValidateProductsRequest)
}

// Validates SKUs and returns matching products keyed by the original map keys.
type ValidateProductsEndpoint struct{}

func (e *ValidateProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ValidateProductsRequest, *apiresource.ValidateProductsResponse] {
	return (&apiendpoint.APIEndpoint[*ValidateProductsRequest, *apiresource.ValidateProductsResponse]{
		Title:             "Validate Products",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/products/actions/validate",
		ContentType:       "application/json",
		Request:           &ValidateProductsRequest{},
		Response:          &apiresource.ValidateProductsResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateProductsRequest) (*apiresource.ValidateProductsResponse, *apierror.APIError) {
			return svc.(ProductSvc).ValidateProducts
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	}).WithDocSource(e)
}
