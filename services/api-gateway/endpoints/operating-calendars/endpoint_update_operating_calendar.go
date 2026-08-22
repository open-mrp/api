package operatingcalendarep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update an operating calendar.
type UpdateOperatingCalendarRequest struct {
	// Operating calendar ID.
	OperatingCalendarID string `path:"id" validate:"required"`
	// Human-readable name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Open weekdays as seven characters of '0' or '1', Monday first. At least one day must be open.
	DaysOfWeek field.Optional[string] `json:"days_of_week,omitzero" validate:"omitempty,len=7"`
	// Local time freight has to be tendered by. Clearing it leaves the ship-by date a day with no time of day attached.
	CutoffAt field.Clearable[string] `json:"cutoff_at,omitzero" validate:"omitempty,max=8"`
	// IANA zone the cutoff is read in. Clearing it on a receiving calendar returns to taking the zone from the ship-to address.
	Timezone field.Clearable[string] `json:"timezone,omitzero" validate:"omitempty,max=64"`
	// Make this the calendar used when nothing more specific is linked. Setting it demotes whichever calendar of the same kind held the role.
	IsDefault field.Optional[bool] `json:"is_default,omitzero"`
}

// Updates an operating calendar.
//
// A calendar's kind cannot change: a shipping calendar that became a receiving one would silently drop the pickup cutoff every commitment resolved against it depends on. Create a second calendar instead.
//
// Changes apply to commitments made from now on. Orders already issued keep the dates they were stamped with, so adding a holiday never retroactively makes a past order late.
type UpdateOperatingCalendarEndpoint struct{}

func (e *UpdateOperatingCalendarEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateOperatingCalendarRequest, *apiresource.OperatingCalendar] {
	return (&apiendpoint.APIEndpoint[*UpdateOperatingCalendarRequest, *apiresource.OperatingCalendar]{
		Title:             "Update Operating Calendar",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/operating-calendars/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError) {
			return svc.(OperatingCalendarSvc).UpdateOperatingCalendar
		},
	})
}
