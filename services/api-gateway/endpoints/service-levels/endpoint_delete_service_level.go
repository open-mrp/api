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
	// The carrier that owns this service level.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
}

// Permanently deletes a service level so it can no longer be selected on shipments.
//
// System-owned service levels and the carrier's default service level cannot be deleted; to remove a default, first clear its `is_default` flag or promote another service level in its place.
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
