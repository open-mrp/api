package operatingcalendarep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create an operating calendar.
type CreateOperatingCalendarRequest struct {
	// Short stable identifier, unique per account.
	Code string `json:"code" validate:"required,max=32"`
	// Human-readable name.
	Name string `json:"name" validate:"required,max=255"`
	// Which side of a shipment this calendar describes.
	Kind constants.OperatingCalendarKind `json:"kind" validate:"required"`
	// Open weekdays as seven characters of '0' or '1', Monday first. "1111100" is Monday to Friday; "1111000" is a Monday-to-Thursday plant. At least one day must be open.
	DaysOfWeek string `json:"days_of_week" validate:"required,len=7"`
	// Local time freight has to be tendered by, as "15:00". Only a shipping calendar accepts one.
	CutoffAt field.Optional[string] `json:"cutoff_at,omitzero" validate:"omitempty,max=8"`
	// IANA zone the cutoff is read in, such as "America/Chicago". On a receiving calendar, leave it unset to take the zone from the ship-to address.
	Timezone field.Optional[string] `json:"timezone,omitzero" validate:"omitempty,max=64"`
	// Make this the calendar used when nothing more specific is linked. Setting it demotes whichever calendar of the same kind held the role.
	IsDefault field.Optional[bool] `json:"is_default,omitzero"`
}

var sampleCreateOperatingCalendarRequest = &CreateOperatingCalendarRequest{
	Code:       "default_ship",
	Name:       "Shipping days",
	Kind:       constants.OperatingCalendarKindShip,
	DaysOfWeek: "1111000",
	CutoffAt:   field.Some("15:00"),
	Timezone:   field.Some("America/Chicago"),
	IsDefault:  field.Some(true),
}

func (*CreateOperatingCalendarRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateOperatingCalendarRequest)
}

// Creates an operating calendar.
//
// The days govern every ship-by date resolved against this calendar: a plant that tenders freight Monday to Thursday never gets committed to a Friday shipment, and a customer's promised delivery date is worked back from a day they can actually receive on.
//
// A calendar starts with no closures. Add holidays and shutdowns to it separately.
type CreateOperatingCalendarEndpoint struct{}

func (e *CreateOperatingCalendarEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateOperatingCalendarRequest, *apiresource.OperatingCalendar] {
	return (&apiendpoint.APIEndpoint[*CreateOperatingCalendarRequest, *apiresource.OperatingCalendar]{
		Title:             "Create Operating Calendar",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/operating-calendars",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateOperatingCalendarRequest) (*apiresource.OperatingCalendar, *apierror.APIError) {
			return svc.(OperatingCalendarSvc).CreateOperatingCalendar
		},
	})
}
