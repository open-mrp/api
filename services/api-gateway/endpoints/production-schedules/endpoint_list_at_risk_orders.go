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

// Request to list the orders a schedule does not build in time.
type ListAtRiskOrdersRequest struct {
	// Schedule version ID.
	ScheduleID string `path:"id" validate:"required"`
}

// Returns the customer commitments this schedule version does not meet, soonest first.
//
// Three ways an order lands here. `past_due` means the constraint stage needed to start before this plan begins. `undated` means the order carries no ship-by commitment at all, so it is treated as owed now. `short` means the plan simply does not build enough of it in time — the campaigns it does allocate are listed alongside, because building three hundred of five hundred is a different conversation from building none.
//
// Read from the version's own record rather than re-solved, so what comes back is what was decided when the plan was made. A version generated before commitments were tracked reports nothing, which is correct: it made no promises it could break.
type ListAtRiskOrdersEndpoint struct{}

func (e *ListAtRiskOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAtRiskOrdersRequest, *apiresource.List[apiresource.ScheduleOrderCoverage]] {
	return (&apiendpoint.APIEndpoint[*ListAtRiskOrdersRequest, *apiresource.List[apiresource.ScheduleOrderCoverage]]{
		Title:             "List Schedule At-Risk Orders",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/at-risk-orders",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeScheduleOrderCoverage,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAtRiskOrdersRequest) (*apiresource.List[apiresource.ScheduleOrderCoverage], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListAtRiskOrders
		},
	})
}
