package operatingcalendarep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to close a calendar on a date.
type CreateOperatingCalendarClosureRequest struct {
	// Operating calendar ID.
	OperatingCalendarID string `path:"id" validate:"required"`
	// The date nothing operates. Truncated to a day.
	ClosedOn time.Time `json:"closed_on" validate:"required"`
	// What the closure is, such as "Thanksgiving Day" or "Summer shutdown".
	Name string `json:"name" validate:"required,max=255"`
}

var sampleCreateOperatingCalendarClosureRequest = &CreateOperatingCalendarClosureRequest{
	OperatingCalendarID: apiresource.SampleOperatingCalendarID,
	ClosedOn:            time.Date(2026, time.November, 26, 0, 0, 0, 0, time.UTC),
	Name:                "Thanksgiving Day",
}

func (*CreateOperatingCalendarClosureRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateOperatingCalendarClosureRequest)
}

// Closes a calendar on a date.
//
// Every ship-by date resolved against this calendar afterwards walks past the closure: a carrier that does not move on Thanksgiving pushes the day an order has to leave earlier, and a plant shutdown does the same.
//
// Closing the same date twice is a no-op rather than an error, so re-seeding a year is safe and never renames a closure somebody has relabelled.
type CreateOperatingCalendarClosureEndpoint struct{}

func (e *CreateOperatingCalendarClosureEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateOperatingCalendarClosureRequest, *apiresource.OperatingCalendarClosure] {
	return (&apiendpoint.APIEndpoint[*CreateOperatingCalendarClosureRequest, *apiresource.OperatingCalendarClosure]{
		Title:             "Create Operating Calendar Closure",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/operating-calendars/{id}/closures",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateOperatingCalendarClosureRequest) (*apiresource.OperatingCalendarClosure, *apierror.APIError) {
			return svc.(OperatingCalendarSvc).CreateOperatingCalendarClosure
		},
	})
}
