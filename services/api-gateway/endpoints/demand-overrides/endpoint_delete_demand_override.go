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
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteDemandOverrideEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteDemandOverrideRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteDemandOverrideRequest, *object.Deleted]{
		Title:             "Delete Demand Override",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-overrides/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeDemandOverride,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDemandOverrides, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteDemandOverrideRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(DemandOverridesSvc).DeleteDemandOverride, constants.ObjectTypeDemandOverride, "id")
		},
	})
}
