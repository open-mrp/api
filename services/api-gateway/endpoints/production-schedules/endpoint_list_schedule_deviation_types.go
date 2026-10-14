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

// Request to list schedule deviation types.
type ListScheduleDeviationTypesRequest struct{}

// Returns the kinds of hand change a schedule deviation can record.
type ListScheduleDeviationTypesEndpoint struct{}

func (e *ListScheduleDeviationTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListScheduleDeviationTypesRequest, *apiresource.List[apiresource.ScheduleDeviationType]] {
	return (&apiendpoint.APIEndpoint[*ListScheduleDeviationTypesRequest, *apiresource.List[apiresource.ScheduleDeviationType]]{
		Title:             "List Schedule Deviation Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/schedule-deviation-types",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeScheduleDeviationType,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListScheduleDeviationTypesRequest) (*apiresource.List[apiresource.ScheduleDeviationType], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListScheduleDeviationTypes
		},
	})
}
