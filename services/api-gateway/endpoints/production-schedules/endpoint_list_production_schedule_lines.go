package productionscheduleep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list the campaigns on a production schedule.
type ListProductionScheduleLinesRequest struct {
	// ID of the schedule version.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Only return campaigns on these machines.
	MachineIDs []string `query:"machine_ids"`
	// Only return campaigns in this horizon week, zero-based.
	WeekIndex *int32 `query:"week_index" validate:"omitempty,min=0"`
}

// Returns the planned campaigns for a schedule version, in the order they run.
type ListProductionScheduleLinesEndpoint struct{}

func (e *ListProductionScheduleLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionScheduleLinesRequest, *apiresource.List[apiresource.ProductionScheduleLine]] {
	return (&apiendpoint.APIEndpoint[*ListProductionScheduleLinesRequest, *apiresource.List[apiresource.ProductionScheduleLine]]{
		Title:             "List Production Schedule Lines",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/lines",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleLine,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionScheduleLinesRequest) (*apiresource.List[apiresource.ProductionScheduleLine], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListProductionScheduleLines
		},
	})
}
