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
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return service levels with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
	// The carrier whose service levels are listed.
	CarrierID string `path:"carrier_id" validate:"required"`
}

// Returns a list of a carrier's service levels.
//
// Sorted by creation date, newest first.
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
		ObjectType: constants.ObjectTypeServiceLevel,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListServiceLevelsRequest) (*apiresource.List[apiresource.ServiceLevel], *apierror.APIError) {
			return svc.(ServiceLevelSvc).ListServiceLevels
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"carrier", "owner", "owner.account"},
		}),
	})
}
