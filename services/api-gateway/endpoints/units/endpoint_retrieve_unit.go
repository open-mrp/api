package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a unit.
type RetrieveUnitRequest struct {
	// Unit ID.
	UnitID string `path:"id" validate:"required"`
}

// Returns a unit by ID, including both account-owned and global system units.
type RetrieveUnitEndpoint struct{}

func (e *RetrieveUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveUnitRequest, *apiresource.Unit] {
	return (&apiendpoint.APIEndpoint[*RetrieveUnitRequest, *apiresource.Unit]{
		Title:               "Retrieve Unit",
		Method:              http.MethodGet,
		Route:               "/v1/catalog/units/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).GetUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
