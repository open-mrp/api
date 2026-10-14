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

// Request to reopen a closed date.
type DeleteOperatingCalendarClosureRequest struct {
	// Operating calendar ID.
	OperatingCalendarID string `path:"id" validate:"required"`
	// Closure ID.
	ClosureID string `path:"closure_id" validate:"required"`
}

// Reopens a date the calendar was closed on.
//
// Used to drop a seeded holiday a plant actually works through. Orders already issued keep the dates they were stamped with.
type DeleteOperatingCalendarClosureEndpoint struct{}

func (e *DeleteOperatingCalendarClosureEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteOperatingCalendarClosureRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteOperatingCalendarClosureRequest, *object.Deleted]{
		Title:             "Delete Operating Calendar Closure",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/operating-calendars/{id}/closures/{closure_id}",
		SuccessStatusCode: http.StatusNoContent,
		ObjectType:        constants.ObjectTypeOperatingCalendarClosure,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteOperatingCalendarClosureRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(OperatingCalendarSvc).DeleteOperatingCalendarClosure, constants.ObjectTypeOperatingCalendarClosure, "closure_id")
		},
	})
}
