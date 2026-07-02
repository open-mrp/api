package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a product type.
type DeleteProductTypeRequest struct {
	// Product type ID.
	ProductTypeID string `path:"id" validate:"required"`
}

// Deletes a product type.
type DeleteProductTypeEndpoint struct{}

func (e *DeleteProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductTypeRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteProductTypeRequest, *apiresource.EmptyResource]{
		Title:               "Delete Product Type",
		Method:              http.MethodDelete,
		Route:               "/v1/catalog/product-types/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductTypes, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductTypeRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductTypeSvc).DeleteProductType
		},
	})
}
