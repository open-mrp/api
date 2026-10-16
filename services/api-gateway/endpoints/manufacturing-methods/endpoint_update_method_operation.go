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

// Request to update an operation.
type UpdateMethodOperationRequest struct {
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
	// Operation ID.
	ID string `path:"id" validate:"required"`
	// Where the operation goes, starting at 1; the others move to make room.
	Position field.Optional[int32] `json:"position,omitzero"`
	// When the operation starts: `after_previous` or `with_previous`.
	Sequence field.Optional[constants.OperationSequence] `json:"sequence,omitzero"`
	// ID of the work center the operation is done at.
	WorkCenterID field.Optional[string] `json:"work_center_id,omitzero"`
	// IDs of the operation's machines afterwards; an empty list removes them all. Archived machines can't be added.
	MachineIDs field.Optional[[]string] `json:"machine_ids,omitzero" validate:"omitempty,max=100"`
	// ID of the scanning station; `null` removes it.
	ScanningStationID field.Clearable[string] `json:"scanning_station_id,omitzero"`
	// Time to set up for one run; `null` removes it.
	SetupTime field.Clearable[apirequest.QuantityInput] `json:"setup_time,omitzero"`
	// Labor time per unit of output.
	LaborTime field.Optional[apirequest.RateInput] `json:"labor_time,omitzero"`
	// Machine time per unit of output; `null` removes it.
	MachineTime field.Clearable[apirequest.RateInput] `json:"machine_time,omitzero"`
	// Cost of labor per unit of time.
	LaborRate field.Optional[apirequest.UnitPriceInput] `json:"labor_rate,omitzero" sensitive:"cost"`
	// Overhead cost per unit of time.
	OverheadRate field.Optional[apirequest.UnitPriceInput] `json:"overhead_rate,omitzero" sensitive:"cost"`
	// Correction to labor time for the pace of the work, as a whole-number percent.
	LevelingPercent field.Optional[string] `json:"leveling_percent,omitzero" format:"decimal"`
	// Allowance added to labor time, as a whole-number percent.
	AllowancePercent field.Optional[string] `json:"allowance_percent,omitzero" format:"decimal"`
	// Instructions for the people doing the operation; `null` removes them. Maximum 10,000 characters.
	WorkInstructions field.Clearable[string] `json:"work_instructions,omitzero" validate:"omitempty,max=10000"`
}

var sampleUpdateMethodOperationRequest = &UpdateMethodOperationRequest{
	Position:        field.Some(int32(2)),
	LevelingPercent: field.Some("20"),
}

func (*UpdateMethodOperationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMethodOperationRequest)
}

// Updates an operation of a draft manufacturing method.
//
// Only the fields you send change. Only a `draft` method can change (409 `invalid_state_transition`).
type UpdateMethodOperationEndpoint struct{}

func (e *UpdateMethodOperationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMethodOperationRequest, *apiresource.MethodOperation] {
	return &apiendpoint.APIEndpoint[*UpdateMethodOperationRequest, *apiresource.MethodOperation]{
		Title:             "Update Method Operation",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             operationsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeMethodOperation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).UpdateMethodOperation
		},
		IncludeConfig: operationIncludes(),
	}
}
