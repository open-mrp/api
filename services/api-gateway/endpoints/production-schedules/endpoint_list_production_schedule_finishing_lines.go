package productionscheduleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list the finishing plan behind a production schedule.
type ListProductionScheduleFinishingLinesRequest struct {
	// ID of the schedule version.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Only the finishing planned for this week, zero-based from the start of the horizon.
	WeekIndex *int32 `query:"week_index" validate:"omitempty,min=0"`
	// Only the finishing planned for this finished good.
	ItemID *string `query:"item_id"`
}

// Returns the second stage of a schedule: how many of which finished good to make from the knitted parts, week by week.
//
// The constraint plan says how much greige to knit and deliberately does not say what to turn it into — a family's demand is pooled onto the greige precisely so the buffer can sit at the undifferentiated stage, where it is cheapest. These lines are where that pooling is undone, against each finished SKU's own stock position, its own orders, and the hours the rest of the factory has that week.
//
// Leveled, not merely allocated. Work that does not fit a week moves to the next one rather than being dropped, so the plan never asks the second stage for more hours than it has. Two things bound it, and they are reported separately in the schedule's diagnostics because they call for opposite responses: a SKU held back for want of greige is a knitting problem, and a SKU held back for want of hours is a finishing one.
//
// Everything is counted in the constraint item's unit, so `greige_consumed` here and `planned_quantity` on the constraint plan are directly comparable — which is what lets the two stages be reconciled rather than only read side by side.
type ListProductionScheduleFinishingLinesEndpoint struct{}

func (e *ListProductionScheduleFinishingLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionScheduleFinishingLinesRequest, *apiresource.List[apiresource.ProductionScheduleFinishingLine]] {
	return (&apiendpoint.APIEndpoint[*ListProductionScheduleFinishingLinesRequest, *apiresource.List[apiresource.ProductionScheduleFinishingLine]]{
		Title:             "List Production Schedule Finishing Lines",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/finishing-lines",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleFinishingLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionScheduleFinishingLinesRequest) (*apiresource.List[apiresource.ProductionScheduleFinishingLine], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListProductionScheduleFinishingLines
		},
	})
}
