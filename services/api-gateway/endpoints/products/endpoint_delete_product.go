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
	"github.com/open-mrp/apikit/object"
)

// Request to delete a product.
type DeleteProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
}

// Soft-deletes a product and returns a deleted stub.
//
// Deletion marks the product's backing item as deleted, so the item and its inventory drop out of catalog and inventory listings too. Deleting the same product again returns an error saying it has already been deleted.
type DeleteProductEndpoint struct{}

func (e *DeleteProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteProductRequest, *object.Deleted]{
		Title:             "Delete Product",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/products/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionDelete}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeProduct,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ProductSvc).DeleteProduct, constants.ObjectTypeProduct, "id")
		},
	})
}
