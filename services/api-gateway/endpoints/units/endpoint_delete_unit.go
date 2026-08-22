package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a unit.
type DeleteUnitRequest struct {
	// Unit ID.
	UnitID string `path:"id" validate:"required"`
}

// Deletes a unit owned by your account.
//
// The unit is also removed from every unit group it belongs to. System units, which are shared across all accounts, cannot be deleted.
type DeleteUnitEndpoint struct{}

func (e *DeleteUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteUnitRequest, *apiresource.EmptyResource]{
		Title:               "Delete Unit",
		Method:              http.MethodDelete,
		Route:               "/v1/catalog/units/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitSvc).DeleteUnit
		},
	})
}
