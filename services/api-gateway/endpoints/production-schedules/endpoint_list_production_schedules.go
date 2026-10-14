package productionscheduleep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list production schedules.
type ListProductionSchedulesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return versions in these lifecycle states.
	//
	// - `draft`: still editable and committed to nothing.
	// - `generating`: the solver is still building the version.
	// - `published`: live, with its first weeks frozen as a commitment to the floor.
	// - `superseded`: a later version was published over the same horizon and replaced this one.
	// - `archived`: retired without being replaced.
	// - `failed`: the solver could not produce a plan.
	Statuses []constants.ProductionScheduleStatus `query:"statuses"`
}

// Returns a paginated list of production schedule versions, newest first.
type ListProductionSchedulesEndpoint struct{}

func (e *ListProductionSchedulesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionSchedulesRequest, *apiresource.List[apiresource.ProductionSchedule]] {
	return (&apiendpoint.APIEndpoint[*ListProductionSchedulesRequest, *apiresource.List[apiresource.ProductionSchedule]]{
		Title:             "List Production Schedules",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionSchedulesRequest) (*apiresource.List[apiresource.ProductionSchedule], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListProductionSchedules
		},
	})
}
