package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list service levels.
type ListServiceLevelsRequest struct {
	apiresource.PaginationRequest
	// The carrier whose service levels are listed.
	CarrierID string `path:"carrier_id" validate:"required"`
}

// Returns a paginated list of the service levels a carrier offers.
//
// Use this rather than the `service_levels` field on the carrier itself when a carrier has more than a handful of services, since that inline list is capped.
type ListServiceLevelsEndpoint struct{}

func (e *ListServiceLevelsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListServiceLevelsRequest, *apiresource.List[apiresource.ServiceLevel]] {
	return (&apiendpoint.APIEndpoint[*ListServiceLevelsRequest, *apiresource.List[apiresource.ServiceLevel]]{
		Title:             "List Service Levels",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		// Service-level reads reuse the carrier relation helper (checkCarrierReadPermission), which requires carriers:read on the own account but customers:read / suppliers:read when an internal actor reads a customer's or supplier's data. Declare the full OR-set so the gateway gate doesn't false-reject those relation-scoped reads.
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCarriers, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		Preview:    true,
		ObjectType: constants.ObjectTypeServiceLevel,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListServiceLevelsRequest) (*apiresource.List[apiresource.ServiceLevel], *apierror.APIError) {
			return svc.(ServiceLevelSvc).ListServiceLevels
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
