package operatingcalendarep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list an account's operating calendars.
type ListOperatingCalendarsRequest struct {
	// Return only shipping or only receiving calendars.
	Kind *constants.OperatingCalendarKind `query:"kind" validate:"omitempty"`
}

// Lists the operating calendars configured for the account.
//
// Both kinds are returned unless `kind` narrows it, ordered with each kind's default first.
type ListOperatingCalendarsEndpoint struct{}

func (e *ListOperatingCalendarsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOperatingCalendarsRequest, *apiresource.List[apiresource.OperatingCalendar]] {
	return (&apiendpoint.APIEndpoint[*ListOperatingCalendarsRequest, *apiresource.List[apiresource.OperatingCalendar]]{
		Title:             "List Operating Calendars",
		Method:            http.MethodGet,
		Route:             "/v1/operations/operating-calendars",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ReadOnly:          true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOperatingCalendarsRequest) (*apiresource.List[apiresource.OperatingCalendar], *apierror.APIError) {
			return svc.(OperatingCalendarSvc).ListOperatingCalendars
		},
	})
}
