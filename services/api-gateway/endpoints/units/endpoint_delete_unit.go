package unitep

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

// Request to delete a unit.
type DeleteUnitRequest struct {
	// Unit ID.
	UnitID string `path:"id" validate:"required"`
}

// Deletes a unit owned by your account.
//
// The unit is also removed from every unit group it belongs to. System units, which are shared across all accounts, cannot be deleted, and neither can a unit that is a unit group's base unit — change the group's base unit or delete the group first — or one that any quantity or price is recorded in.
type DeleteUnitEndpoint struct{}

func (e *DeleteUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteUnitRequest, *object.Deleted]{
		Title:             "Delete Unit",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/units/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeUnit,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(UnitSvc).DeleteUnit, constants.ObjectTypeUnit, "id")
		},
	})
}
