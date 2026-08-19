package operatingcalendarep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an operating calendar.
type DeleteOperatingCalendarRequest struct {
	// Operating calendar ID.
	OperatingCalendarID string `path:"id" validate:"required"`
}

// Deletes an operating calendar.
//
// Refused while any address, customer, customer group, or account setting still points at it. Deleting a calendar out from under its references would quietly return every affected order to a plain Monday-to-Friday week, which reads as the feature breaking rather than as a decision anybody made — so re-point them first.
type DeleteOperatingCalendarEndpoint struct{}

func (e *DeleteOperatingCalendarEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteOperatingCalendarRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteOperatingCalendarRequest, *apiresource.EmptyResource]{
		Title:             "Delete Operating Calendar",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/operating-calendars/{id}",
		SuccessStatusCode: http.StatusNoContent,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteOperatingCalendarRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(OperatingCalendarSvc).DeleteOperatingCalendar
		},
	})
}
