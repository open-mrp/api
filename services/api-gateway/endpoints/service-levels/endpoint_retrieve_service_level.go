package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a service level.
type RetrieveServiceLevelRequest struct {
	// The carrier that owns this service level.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
}

// Returns a service level by ID.
type RetrieveServiceLevelEndpoint struct{}

func (e *RetrieveServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveServiceLevelRequest, *apiresource.ServiceLevel] {
	return (&apiendpoint.APIEndpoint[*RetrieveServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Retrieve Service Level",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		// Service-level reads reuse the carrier relation helper (checkCarrierReadPermission).
		RequiredPermissions:     []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionRead}},
		CounterpartyPermissions: apiendpoint.Counterparties(types.ActionRead),
		Preview:                 true,
		ObjectType:              constants.ObjectTypeServiceLevel,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).GetServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
