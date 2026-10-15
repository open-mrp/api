package workcenterep

import (
	"context"
	"net/http"

	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update a work center.
type UpdateWorkCenterRequest struct {
	// ID of the work center to update.
	WorkCenterID string `path:"id" validate:"required"`
	// Display name of the work center.
	//
	// Unique within your account (case-insensitive); maximum 255 characters.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// ID of the location where the work center operates.
	//
	// The location must be active.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// IDs of scanning stations to move into this work center.
	//
	// Adds to the work center's stations: listed stations move in, and its other stations stay.
	ScanningStationIDs []string `json:"scanning_station_ids,omitzero" validate:"omitempty,max=100"`
	// IDs of machines to move into this work center.
	//
	// Adds to the work center's machines: listed machines move in, and its other machines stay. Archived machines can't be moved.
	MachineIDs []string `json:"machine_ids,omitzero" validate:"omitempty,max=100"`
	// Hourly labor rate for work done at this work center.
	//
	// `per_unit_id` must be a time unit. A labor rate can be changed but not removed.
	LaborRate field.Optional[apirequest.UnitPriceInput] `json:"labor_rate,omitzero" sensitive:"cost"`
}

var sampleUpdateWorkCenterRequest = &UpdateWorkCenterRequest{
	Name: field.Some("Assembly"),
}

func (*UpdateWorkCenterRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateWorkCenterRequest)
}

// Updates a work center.
//
// Only the fields you send change. Listed machines and scanning stations move into the work center; its other members stay. Returns a conflict error if another work center already has the new name.
type UpdateWorkCenterEndpoint struct{}

func (e *UpdateWorkCenterEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateWorkCenterRequest, *apiresource.WorkCenter] {
	return (&apiendpoint.APIEndpoint[*UpdateWorkCenterRequest, *apiresource.WorkCenter]{
		Title:             "Update Work Center",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/work-centers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError) {
			return svc.(WorkCenterSvc).UpdateWorkCenter
		},
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionUpdate},
		}},
		ObjectType: constants.ObjectTypeWorkCenter,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeWorkCenter,
			Fields:     workCenterIncludes,
		}),
	})
}
