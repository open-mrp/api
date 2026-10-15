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

// Deletes a product line.
//
// Only lines your account created can be deleted, and only while no product, price-list rule, sales target or territory uses them. Archive a line that is in use by setting `status` to `archived`. Customer access to the line is removed with it.
type DeleteProductLineEndpoint struct{}

func (e *DeleteProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductLineRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteProductLineRequest, *object.Deleted]{
		Title:             "Delete Product Line",
		Method:            http.MethodDelete,
		Route:             productLineRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeProductLine,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductLineRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ProductLineSvc).DeleteProductLine, constants.ObjectTypeProductLine, "id")
		},
	})
}
