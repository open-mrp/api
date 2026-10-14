package productlineep

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

// Request to delete a product line.
type DeleteProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
}

// Permanently deletes a product line your account owns.
//
// The reserved `shipping`, `service`, `credit`, and `tax` lines cannot be deleted, and neither can the shared system lines, which belong to no single account. Deleting a line that was already deleted returns an already-deleted error rather than succeeding silently.
type DeleteProductLineEndpoint struct{}

func (e *DeleteProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductLineRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteProductLineRequest, *object.Deleted]{
		Title:             "Delete Product Line",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/product-lines/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductLineRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ProductLineSvc).DeleteProductLine, constants.ObjectTypeProductLine, "id")
		},
	})
}
