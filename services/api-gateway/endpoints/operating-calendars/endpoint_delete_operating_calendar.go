package operatingcalendarep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteOperatingCalendarEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteOperatingCalendarRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteOperatingCalendarRequest, *object.Deleted]{
		Title:             "Delete Operating Calendar",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/operating-calendars/{id}",
		SuccessStatusCode: http.StatusNoContent,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteOperatingCalendarRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(OperatingCalendarSvc).DeleteOperatingCalendar, constants.ObjectTypeOperatingCalendar, "id")
		},
	})
}
