package machinedowntimeep

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

// Request to retrieve a machine downtime event.
type RetrieveMachineDowntimeEventRequest struct {
	// ID of the downtime event to retrieve.
	MachineDowntimeEventID string `path:"id" validate:"required"`
}

// Returns a single machine downtime event.
type RetrieveMachineDowntimeEventEndpoint struct{}

func (e *RetrieveMachineDowntimeEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent] {
	return (&apiendpoint.APIEndpoint[*RetrieveMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent]{
		Title:             "Retrieve Machine Downtime Event",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeMachineDowntimeEvent,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError) {
			return svc.(MachineDowntimeSvc).GetDowntimeEvent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachineDowntimeEvent,
			Fields:     []string{"machine", "department", "item", "reported_by"},
		}),
	})
}
