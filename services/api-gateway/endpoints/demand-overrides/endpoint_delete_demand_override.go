package demandoverridesep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a demand override.
type DeleteDemandOverrideRequest struct {
	// ID of the demand override.
	DemandOverrideID string `path:"id" validate:"required"`
}

// Deletes a demand override permanently.
//
// Schedules that have already been generated are unaffected: each one records the overrides it applied, so deleting an override changes only schedules generated from now on. To stop an override applying while keeping it on file, deactivate it instead.
type DeleteDemandOverrideEndpoint struct{}

func (e *DeleteDemandOverrideEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteDemandOverrideRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteDemandOverrideRequest, *apiresource.EmptyResource]{
		Title:             "Delete Demand Override",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-overrides/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeDemandOverride,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDemandOverrides, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteDemandOverrideRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(DemandOverridesSvc).DeleteDemandOverride
		},
	})
}
