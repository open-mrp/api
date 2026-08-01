package productionscheduleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the item policies behind a production schedule.
type ListProductionScheduleItemPoliciesRequest struct {
	// ID of the schedule version.
	ProductionScheduleID string `path:"id" validate:"required"`
}

// Returns the per-item policy behind a schedule version, ordered by constraint run hours descending.
//
// This is the "why" behind every campaign: lot size, reorder point, safety stock and lead times as they stood when the plan was generated.
type ListProductionScheduleItemPoliciesEndpoint struct{}

func (e *ListProductionScheduleItemPoliciesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionScheduleItemPoliciesRequest, *apiresource.List[apiresource.ProductionScheduleItemPolicy]] {
	return (&apiendpoint.APIEndpoint[*ListProductionScheduleItemPoliciesRequest, *apiresource.List[apiresource.ProductionScheduleItemPolicy]]{
		Title:             "List Production Schedule Item Policies",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/item-policies",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleItemPolicy,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionScheduleItemPoliciesRequest) (*apiresource.List[apiresource.ProductionScheduleItemPolicy], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListProductionScheduleItemPolicies
		},
	})
}
