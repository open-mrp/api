package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list units.
type ListUnitsRequest struct {
	apiresource.PaginationRequest
	// Filter by unit dimension.
	Type *constants.UnitType `query:"type"`
	// Return only units that belong to at least one of the given unit groups.
	UnitGroupIDs []string `query:"unit_group_ids"`
}

// Returns a paginated list of units for the current account, including both account-owned and global system units.
type ListUnitsEndpoint struct{}

func (e *ListUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUnitsRequest, *apiresource.List[apiresource.Unit]] {
	return (&apiendpoint.APIEndpoint[*ListUnitsRequest, *apiresource.List[apiresource.Unit]]{
		Title:               "List Units",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/units",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListUnitsRequest) (*apiresource.List[apiresource.Unit], *apierror.APIError) {
			return svc.(UnitSvc).ListUnits
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
