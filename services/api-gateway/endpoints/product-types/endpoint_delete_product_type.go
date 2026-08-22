package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a product type.
type DeleteProductTypeRequest struct {
	// Product type ID.
	ProductTypeID string `path:"id" validate:"required"`
}

// Deletes a product type.
//
// Products point at their product type by code, and nothing blocks the delete, so removing a type that products still use leaves those products referencing a code that no longer resolves. Reassign or delete those products first. Product types are shared across all accounts, so the deletion affects every account.
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
