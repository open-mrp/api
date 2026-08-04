package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to look up products by SKU.
type ValidateProductsRequest struct {
	// Map of caller-chosen keys to SKU values to look up.
	//
	// SKUs are matched case-insensitively. Each key is echoed back in the response with its matched product; keys whose SKU does not match any product are omitted.
	ProductsMap map[string]string `json:"products_map" validate:"required"`
}

var sampleValidateProductsRequest = &ValidateProductsRequest{
	ProductsMap: map[string]string{"0": apiresource.SampleItemSKU},
}

func (*ValidateProductsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleValidateProductsRequest)
}

// Resolves a batch of SKUs to products in one call, keyed by the keys you supplied.
//
// Useful before importing order lines from a spreadsheet or a customer document: send each row's SKU under its row key and check which keys come back. Unmatched SKUs are simply left out of the response rather than reported as errors, and unlike the product list this covers products of every type, not just `sale`.
type ValidateProductsEndpoint struct{}

func (e *ValidateProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ValidateProductsRequest, *apiresource.ValidateProductsResponse] {
	return (&apiendpoint.APIEndpoint[*ValidateProductsRequest, *apiresource.ValidateProductsResponse]{
		Title:               "Validate Products",
		Method:              http.MethodPut,
		Route:               "/v1/catalog/products/actions/validate",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		ObjectType:          constants.ObjectTypeProduct,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateProductsRequest) (*apiresource.ValidateProductsResponse, *apierror.APIError) {
			return svc.(ProductSvc).ValidateProducts
		},
		IncludeConfig: validateProductsIncludeConfig(),
	})
}

func validateProductsIncludeConfig() *apiendpoint.IncludeConfig {
	cfg := apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeProduct,
		Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
	})
	cfg.ExtractRoots = func(resp any) []any {
		vpr := resp.(*apiresource.ValidateProductsResponse)
		roots := make([]any, 0, len(vpr.Products))
		for _, p := range vpr.Products {
			if p != nil {
				roots = append(roots, p)
			}
		}
		return roots
	}
	return cfg
}
