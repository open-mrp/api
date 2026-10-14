package servicelevelep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list service levels.
type ListServiceLevelsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
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
		// Service-level reads reuse the carrier relation helper (checkCarrierReadPermission).
		Auth:       authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
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
