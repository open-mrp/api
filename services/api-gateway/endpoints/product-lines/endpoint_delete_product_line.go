package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a product line.
type DeleteProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
}

// Permanently deletes a product line your account owns.
//
// The reserved `shipping`, `service`, `credit`, and `tax` lines cannot be deleted, and neither can the shared system lines, which belong to no single account. Deleting a line that was already deleted returns an already-deleted error rather than succeeding silently.
type DeleteProductLineEndpoint struct{}

func (e *DeleteProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductLineRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteProductLineRequest, *apiresource.EmptyResource]{
		Title:               "Delete Product Line",
		Method:              http.MethodDelete,
		Route:               "/v1/catalog/product-lines/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductLineSvc).DeleteProductLine
		},
	})
}
