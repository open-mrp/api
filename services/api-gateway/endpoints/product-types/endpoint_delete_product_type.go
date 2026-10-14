package producttypeep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductTypeRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteProductTypeRequest, *object.Deleted]{
		Title:             "Delete Product Type",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/product-types/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeProductType,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductTypes, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductTypeRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ProductTypeSvc).DeleteProductType, constants.ObjectTypeProductType, "id")
		},
	})
}
