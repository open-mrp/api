package manufacturingmethodep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/field"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to add an operation to a draft manufacturing method.
type CreateMethodOperationRequest struct {
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
	// Where the operation goes, starting at 1; later operations move down one. Defaults to after the last operation.
	Position field.Optional[int32] `json:"position,omitzero"`
	// When the operation starts: `after_previous` (the default) or `with_previous`, alongside the previous operation.
	Sequence field.Optional[constants.OperationSequence] `json:"sequence,omitzero"`
	// ID of the work center the operation is done at.
	WorkCenterID string `json:"work_center_id" validate:"required"`
	// IDs of the machines the operation runs on. Archived machines can't be added.
	MachineIDs []string `json:"machine_ids,omitzero" validate:"omitempty,max=100"`
	// ID of the scanning station the operation's batches are scanned at.
	ScanningStationID field.Optional[string] `json:"scanning_station_id,omitzero"`
	// Time to set up for one run, in a time unit.
	SetupTime field.Optional[apirequest.QuantityInput] `json:"setup_time,omitzero"`
	// Labor time per unit of output: a time unit per the output's unit, such as hours per pair.
	LaborTime apirequest.RateInput `json:"labor_time" validate:"required"`
	// Machine time per unit of output, like `labor_time`.
	MachineTime field.Optional[apirequest.RateInput] `json:"machine_time,omitzero"`
	// Cost of labor per unit of time. Defaults to the work center's labor rate; required when the work center has none.
	LaborRate field.Optional[apirequest.UnitPriceInput] `json:"labor_rate,omitzero" sensitive:"cost"`
	// Overhead cost per unit of time. Defaults to zero, per the labor rate's unit.
	OverheadRate field.Optional[apirequest.UnitPriceInput] `json:"overhead_rate,omitzero" sensitive:"cost"`
	// Correction to labor time for the pace of the work, as a whole-number percent. Defaults to `"0"`.
	LevelingPercent field.Optional[string] `json:"leveling_percent,omitzero" format:"decimal"`
	// Allowance added to labor time for breaks and fatigue, as a whole-number percent. Defaults to `"0"`.
	AllowancePercent field.Optional[string] `json:"allowance_percent,omitzero" format:"decimal"`
	// Instructions for the people doing the operation. Maximum 10,000 characters.
	WorkInstructions field.Optional[string] `json:"work_instructions,omitzero" validate:"omitempty,max=10000"`
}

var sampleCreateMethodOperationRequest = &CreateMethodOperationRequest{
	WorkCenterID:     apiresource.SampleWorkCenterID,
	MachineIDs:       []string{apiresource.SampleMachineID},
	LaborTime:        apirequest.RateInput{Value: "0.02", UnitID: apiresource.SampleUnitID, PerUnitID: apiresource.SampleUnitID},
	LevelingPercent:  field.Some("15"),
	AllowancePercent: field.Some("10"),
}

func (*CreateMethodOperationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMethodOperationRequest)
}

// Adds an operation to a draft manufacturing method.
//
// Only a `draft` method can change (409 `invalid_state_transition`). A method has at most 100 operations.
type CreateMethodOperationEndpoint struct{}

func (e *CreateMethodOperationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMethodOperationRequest, *apiresource.MethodOperation] {
	return &apiendpoint.APIEndpoint[*CreateMethodOperationRequest, *apiresource.MethodOperation]{
		Title:             "Create Method Operation",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             operationsRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeMethodOperation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).CreateMethodOperation
		},
		IncludeConfig: operationIncludes(),
	}
}
