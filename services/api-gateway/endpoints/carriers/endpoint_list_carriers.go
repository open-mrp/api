package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list carriers.
type ListCarriersRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of the carriers available to the current account.
//
// This covers the carriers you have created plus the platform-provided system carriers that every account shares.
type ListCarriersEndpoint struct{}

func (e *ListCarriersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCarriersRequest, *apiresource.List[apiresource.Carrier]] {
	return (&apiendpoint.APIEndpoint[*ListCarriersRequest, *apiresource.List[apiresource.Carrier]]{
		Title:             "List Carriers",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCarriers, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		Preview:    true,
		ObjectType: constants.ObjectTypeCarrier,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCarriersRequest) (*apiresource.List[apiresource.Carrier], *apierror.APIError) {
			return svc.(CarrierSvc).ListCarriers
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"owner", "owner.account", "service_levels"},
		}),
	})
}
