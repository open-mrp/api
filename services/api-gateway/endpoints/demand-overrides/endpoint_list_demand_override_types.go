package demandoverridesep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list demand override types.
type ListDemandOverrideTypesRequest struct{}

// Returns the demand override types, which describe how an override's value adjusts the forecast.
type ListDemandOverrideTypesEndpoint struct{}

func (e *ListDemandOverrideTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListDemandOverrideTypesRequest, *apiresource.List[apiresource.DemandOverrideType]] {
	return (&apiendpoint.APIEndpoint[*ListDemandOverrideTypesRequest, *apiresource.List[apiresource.DemandOverrideType]]{
		Title:             "List Demand Override Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-override-types",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeDemandOverrideType,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDemandOverrides, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDemandOverrideTypesRequest) (*apiresource.List[apiresource.DemandOverrideType], *apierror.APIError) {
			return svc.(DemandOverridesSvc).ListDemandOverrideTypes
		},
	})
}
