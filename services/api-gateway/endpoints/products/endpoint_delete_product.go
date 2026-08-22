package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a product.
type DeleteProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
}

// Soft-deletes a product and returns it as it stood at deletion.
//
// Deletion marks the product's backing item as deleted, so the item and its inventory drop out of catalog and inventory listings too. Deleting the same product again returns an error saying it has already been deleted.
type DeleteProductEndpoint struct{}

func (e *DeleteProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*DeleteProductRequest, *apiresource.Product]{
		Title:               "Delete Product",
		Method:              http.MethodDelete,
		Route:               "/v1/catalog/products/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionDelete}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeProduct,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).DeleteProduct
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
