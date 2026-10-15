package productep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a product.
type RetrieveProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
}

// Returns a product by ID.
type RetrieveProductEndpoint struct{}

func (e *RetrieveProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductRequest, *apiresource.Product]{
		Title:             "Retrieve Product",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/products/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).GetProduct
		},
		ObjectType: constants.ObjectTypeProduct,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.attributes"},
		}),
	})
}
