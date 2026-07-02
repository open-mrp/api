package servicelevelep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a service level.
type DeleteServiceLevelRequest struct {
	// Carrier ID.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
}

// Permanently deletes a service level.
//
// System-owned service levels and the carrier's default service level cannot be deleted; unset `is_default` first to delete a default.
type DeleteServiceLevelEndpoint struct{}

func (e *DeleteServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteServiceLevelRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteServiceLevelRequest, *apiresource.EmptyResource]{
		Title:               "Delete Service Level",
		Method:              http.MethodDelete,
		Route:               "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteServiceLevelRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ServiceLevelSvc).DeleteServiceLevel
		},
	})
}
