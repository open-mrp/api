package operatingcalendarep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve one operating calendar.
type RetrieveOperatingCalendarRequest struct {
	// Operating calendar ID.
	OperatingCalendarID string `path:"id" validate:"required"`
}

// Retrieves one operating calendar by ID.
type RetrieveOperatingCalendarEndpoint struct{}

func (e *RetrieveOperatingCalendarEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveOperatingCalendarRequest, *apiresource.OperatingCalendar] {
	return (&apiendpoint.APIEndpoint[*RetrieveOperatingCalendarRequest, *apiresource.OperatingCalendar]{
		Title:             "Retrieve Operating Calendar",
		Method:            http.MethodGet,
		Route:             "/v1/operations/operating-calendars/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ReadOnly:          true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError) {
			return svc.(OperatingCalendarSvc).RetrieveOperatingCalendar
		},
	})
}
