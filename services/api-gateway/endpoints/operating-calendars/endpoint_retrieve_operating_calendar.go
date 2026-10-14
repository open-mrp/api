package operatingcalendarep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError) {
			return svc.(OperatingCalendarSvc).RetrieveOperatingCalendar
		},
	})
}
