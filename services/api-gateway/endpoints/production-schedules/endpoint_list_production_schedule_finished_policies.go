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

// Request to list the finished-goods targets behind a production schedule.
type ListProductionScheduleFinishedPoliciesRequest struct {
	// ID of the schedule version.
	ProductionScheduleID string `path:"id" validate:"required"`
}

// Returns the per-finished-SKU inventory targets behind a schedule version, grouped under the constraint item each one is made from.
//
// The item policies pool every finished good a constraint item feeds into one echelon figure, which is what the build decision is made against. These rows are what that pooling hides: each finished SKU's own demand, its own variability, its own stock, and a buffer sized against the finishing lead time rather than the constraint's.
//
// The two stages do not overlap, so together they describe the whole network's stock without counting any of it twice: the constraint stage holds its pooled buffer, and the finished stage holds these.
type ListProductionScheduleFinishedPoliciesEndpoint struct{}

func (e *ListProductionScheduleFinishedPoliciesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionScheduleFinishedPoliciesRequest, *apiresource.List[apiresource.ProductionScheduleFinishedPolicy]] {
	return (&apiendpoint.APIEndpoint[*ListProductionScheduleFinishedPoliciesRequest, *apiresource.List[apiresource.ProductionScheduleFinishedPolicy]]{
		Title:             "List Production Schedule Finished Policies",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/finished-policies",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleFinishedPolicy,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionScheduleFinishedPoliciesRequest) (*apiresource.List[apiresource.ProductionScheduleFinishedPolicy], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListProductionScheduleFinishedPolicies
		},
	})
}
