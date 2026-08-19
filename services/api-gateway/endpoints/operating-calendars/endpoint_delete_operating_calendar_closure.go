package operatingcalendarep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
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

func (e *DeleteOperatingCalendarClosureEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteOperatingCalendarClosureRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteOperatingCalendarClosureRequest, *apiresource.EmptyResource]{
		Title:             "Delete Operating Calendar Closure",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/operating-calendars/{id}/closures/{closure_id}",
		SuccessStatusCode: http.StatusNoContent,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteOperatingCalendarClosureRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(OperatingCalendarSvc).DeleteOperatingCalendarClosure
		},
	})
}
