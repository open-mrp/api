package servicelevelep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a service level.
type DeleteServiceLevelRequest struct {
	// The carrier that owns this service level.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
}

// Deletes a service level.
//
// Only while no order, shipment or shipping term uses it, and it isn't its carrier's default. Archive it instead by setting `status` to `archived`.
type DeleteServiceLevelEndpoint struct{}

func (e *DeleteServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteServiceLevelRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteServiceLevelRequest, *object.Deleted]{
		Title:             "Delete Service Level",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeServiceLevel,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteServiceLevelRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ServiceLevelSvc).DeleteServiceLevel, constants.ObjectTypeServiceLevel, "id")
		},
	})
}
