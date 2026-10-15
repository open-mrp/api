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

// Request to create a work center.
type CreateWorkCenterRequest struct {
	// Display name of the work center.
	//
	// Unique within your account (case-insensitive); maximum 255 characters.
	Name string `json:"name" validate:"required,max=255"`
	// ID of the location where the work center operates.
	//
	// The location must be active.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// IDs of scanning stations to move into this work center.
	//
	// A scanning station belongs to one work center at a time, so each listed station leaves its current one.
	ScanningStationIDs []string `json:"scanning_station_ids,omitzero" validate:"omitempty,max=100"`
	// IDs of machines to move into this work center.
	//
	// A machine belongs to one work center at a time, so each listed machine leaves its current one. Archived machines can't be moved.
	MachineIDs []string `json:"machine_ids,omitzero" validate:"omitempty,max=100"`
	// Hourly labor rate for work done at this work center, such as `{"amount": "45.00", "currency": "usd", "per_unit_id": "<hour unit ID>"}`.
	//
	// `per_unit_id` must be a time unit.
	LaborRate field.Optional[apirequest.UnitPriceInput] `json:"labor_rate,omitzero" sensitive:"cost"`
}

var sampleCreateWorkCenterRequest = &CreateWorkCenterRequest{
	Name:               apiresource.SampleWorkCenterName,
	ScanningStationIDs: []string{apiresource.SampleScanningStationID},
	MachineIDs:         []string{apiresource.SampleMachineID},
}

func (*CreateWorkCenterRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateWorkCenterRequest)
}

// Creates a work center.
//
// Listed machines and scanning stations move into it. Returns a conflict error if another work center already has the name.
type CreateWorkCenterEndpoint struct{}

func (e *CreateWorkCenterEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateWorkCenterRequest, *apiresource.WorkCenter] {
	return (&apiendpoint.APIEndpoint[*CreateWorkCenterRequest, *apiresource.WorkCenter]{
		Title:             "Create Work Center",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/work-centers",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError) {
			return svc.(WorkCenterSvc).CreateWorkCenter
		},
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionCreate},
		}},
		LocationFunc: func(resp *apiresource.WorkCenter) string {
			return "/v1/operations/work-centers/" + resp.ID
		},
		ObjectType: constants.ObjectTypeWorkCenter,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeWorkCenter,
			Fields:     workCenterIncludes,
		}),
	})
}
