package demandoverridesep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list demand override types.
type ListDemandOverrideTypesRequest struct{}

// Returns the demand override types, which describe how an override's value adjusts the forecast.
//
// The taxonomy is platform-provided and identical for every account; each type's `code` is a value accepted as an override's `adjustment`.
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
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDemandOverrides, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDemandOverrideTypesRequest) (*apiresource.List[apiresource.DemandOverrideType], *apierror.APIError) {
			return svc.(DemandOverridesSvc).ListDemandOverrideTypes
		},
	})
}
