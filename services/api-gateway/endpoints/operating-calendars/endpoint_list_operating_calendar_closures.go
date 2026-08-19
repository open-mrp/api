package operatingcalendarep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list a calendar's closures.
type ListOperatingCalendarClosuresRequest struct {
	// Operating calendar ID.
	OperatingCalendarID string `path:"id" validate:"required"`
	// Earliest closure date to return. Defaults to a year ago.
	FromDate *time.Time `query:"from_date" validate:"omitempty"`
	// Latest closure date to return. Defaults to a year ahead.
	ToDate *time.Time `query:"to_date" validate:"omitempty"`
}

// Lists the dates a calendar is shut, within a date window.
//
// Bounded rather than exhaustive: a calendar accumulates closures indefinitely, and the useful answer is the year either side of today. Widen it with `from_date` and `to_date`.
type ListOperatingCalendarClosuresEndpoint struct{}

func (e *ListOperatingCalendarClosuresEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOperatingCalendarClosuresRequest, *apiresource.List[apiresource.OperatingCalendarClosure]] {
	return (&apiendpoint.APIEndpoint[*ListOperatingCalendarClosuresRequest, *apiresource.List[apiresource.OperatingCalendarClosure]]{
		Title:             "List Operating Calendar Closures",
		Method:            http.MethodGet,
		Route:             "/v1/operations/operating-calendars/{id}/closures",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ReadOnly:          true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOperatingCalendarClosuresRequest) (*apiresource.List[apiresource.OperatingCalendarClosure], *apierror.APIError) {
			return svc.(OperatingCalendarSvc).ListOperatingCalendarClosures
		},
	})
}
