package machinedowntimeep

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

// Request to list machine downtime events.
type ListMachineDowntimeEventsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return events for these machines.
	MachineIDs []string `query:"machine_ids"`
	// Only return events for machines in these departments.
	DepartmentIDs []string `query:"department_ids"`
	// Only return events logged against these reasons.
	Reasons []constants.MachineDowntimeReasonCode `query:"reasons"`
	// Only return events that are still open, meaning the machine is down right now.
	//
	// Sending `false` is the same as leaving it out: both open and closed events come back.
	Open bool `query:"open"`
	// Only return events that started on or after this timestamp, formatted as RFC3339.
	StartDate *string `query:"starts_at"`
	// Only return events that started on or before this timestamp, formatted as RFC3339.
	EndDate *string `query:"ends_at"`
}

// Returns a paginated list of machine downtime events, most recently started first.
//
// The search term matches text in the event note. Filters combine, so a machine, a reason and a date range narrow the list together.
type ListMachineDowntimeEventsEndpoint struct{}

func (e *ListMachineDowntimeEventsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMachineDowntimeEventsRequest, *apiresource.List[apiresource.MachineDowntimeEvent]] {
	return (&apiendpoint.APIEndpoint[*ListMachineDowntimeEventsRequest, *apiresource.List[apiresource.MachineDowntimeEvent]]{
		Title:             "List Machine Downtime Events",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeMachineDowntimeEvent,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMachineDowntimeEventsRequest) (*apiresource.List[apiresource.MachineDowntimeEvent], *apierror.APIError) {
			return svc.(MachineDowntimeSvc).ListDowntimeEvents
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachineDowntimeEvent,
			Fields:     []string{"machine", "department", "item", "reported_by"},
		}),
	})
}
