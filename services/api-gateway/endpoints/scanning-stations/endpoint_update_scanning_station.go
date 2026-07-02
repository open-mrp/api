package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a scanning station.
type UpdateScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
	// Display name of the scanning station.
	//
	// Must be unique within your account; maximum 255 characters.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Free-form notes about the scanning station.
	//
	// Send `null` to clear.
	Notes field.Clearable[string] `json:"notes,omitzero"`
	// Size of the labels printed at this station, given as width-by-height (for example, `1x1`).
	LabelSizeCode field.Optional[constants.LabelSizeCode] `json:"label_size,omitzero"`
	// Type of label printed at this station.
	//
	// - `tag`: a label attached to the physical product.
	// - `traveler`: a routing sheet that accompanies the batch through every production step.
	LabelTypeCode field.Optional[constants.LabelTypeCode] `json:"label_type,omitzero"`
	// Whether operators must perform a material check at this station.
	//
	// - `none`: no additional operator check is required.
	// - `material_check`: a material check is expected before the operation.
	OperatorRequirement field.Optional[constants.OperatorRequirement] `json:"operator_requirement,omitzero"`
}

var sampleUpdateScanningStationName = "Station B"
var sampleUpdateScanningStationNotes = "Relocated to the finishing area."
var sampleUpdateScanningStationRequest = &UpdateScanningStationRequest{
	Name:                field.Some(sampleUpdateScanningStationName),
	Notes:               field.Set(sampleUpdateScanningStationNotes),
	LabelSizeCode:       field.Some(constants.LabelSizeCodeOneByOne),
	LabelTypeCode:       field.Some(constants.LabelTypeCodeTag),
	OperatorRequirement: field.Some(constants.OperatorRequirementMaterialCheck),
}

func (*UpdateScanningStationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateScanningStationRequest)
}

// Partially updates a scanning station.
//
// Only the fields provided in the request are changed. Returns a conflict error if the new name is already in use by another scanning station.
type UpdateScanningStationEndpoint struct{}

func (e *UpdateScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateScanningStationRequest, *apiresource.ScanningStation] {
	return (&apiendpoint.APIEndpoint[*UpdateScanningStationRequest, *apiresource.ScanningStation]{
		Title:               "Update Scanning Station",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/operations/scanning-stations/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeScanningStation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
			return svc.(ScanningStationSvc).UpdateScanningStation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeScanningStation,
			Fields:     []string{"department", "production_steps"},
		}),
	})
}
