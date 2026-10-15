package scanningstationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update a scanning station.
//
// The station's type is set at creation and can't be changed.
type UpdateScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
	// Display name of the scanning station.
	//
	// Unique within your account (case-insensitive); maximum 255 characters.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Size of the labels printed at this station, given as width-by-height (for example, `1x1`).
	//
	// Send `null` or an empty string to clear.
	LabelSizeCode field.Clearable[constants.LabelSizeCode] `json:"label_size,omitzero"`
	// Type of label printed at this station.
	//
	// - `tag`: a label attached to the physical product.
	// - `traveler`: a routing sheet that accompanies the batch through every production step.
	//
	// Send `null` or an empty string to clear.
	LabelTypeCode field.Clearable[constants.LabelTypeCode] `json:"label_type,omitzero"`
	// Whether operators must perform a material check at this station.
	//
	// - `none`: no additional operator check is required.
	// - `material_check`: a material check is expected before the operation.
	OperatorRequirement field.Optional[constants.OperatorRequirement] `json:"operator_requirement,omitzero"`
	// ID of the work center to move the station to.
	WorkCenterID field.Optional[string] `json:"work_center_id,omitzero" validate:"omitempty"`
}

var sampleUpdateScanningStationName = "Station B"
var sampleUpdateScanningStationRequest = &UpdateScanningStationRequest{
	Name:                field.Some(sampleUpdateScanningStationName),
	LabelSizeCode:       field.Set(constants.LabelSizeCodeOneByOne),
	LabelTypeCode:       field.Set(constants.LabelTypeCodeTag),
	OperatorRequirement: field.Some(constants.OperatorRequirementMaterialCheck),
}

func (*UpdateScanningStationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateScanningStationRequest)
}

// Updates a scanning station.
//
// Only the fields you send change. Returns a conflict error if another scanning station already has the new name.
type UpdateScanningStationEndpoint struct{}

func (e *UpdateScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateScanningStationRequest, *apiresource.ScanningStation] {
	return (&apiendpoint.APIEndpoint[*UpdateScanningStationRequest, *apiresource.ScanningStation]{
		Title:             "Update Scanning Station",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeScanningStation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
			return svc.(ScanningStationSvc).UpdateScanningStation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeScanningStation,
			Fields:     []string{"work_center", "production_steps"},
		}),
	})
}
