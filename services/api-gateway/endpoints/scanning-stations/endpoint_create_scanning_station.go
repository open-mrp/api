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

// Request to create a scanning station.
type CreateScanningStationRequest struct {
	// Display name of the scanning station.
	//
	// Must be unique within your account; maximum 255 characters.
	Name string `json:"name" validate:"required,max=255"`
	// Free-form notes about the scanning station.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Scanning station type, determining which batch operation an operator performs when they scan here.
	//
	// - `init_batch`: starts a new batch at the beginning of a production flow.
	// - `merge_batch`: combines several scanned batches into one.
	// - `move_batch`: advances a batch through a production step connected to this station.
	// - `split_batch`: divides a batch into several batches.
	//
	// The type cannot be changed after creation.
	Type constants.ScanningStationType `json:"type" validate:"required"`
	// Whether operators must perform a material check at this station.
	//
	// - `none`: no additional operator check is required.
	// - `material_check`: a material check is expected before the operation.
	OperatorRequirement constants.OperatorRequirement `json:"operator_requirement" validate:"required"`
	// ID of the department this station belongs to.
	//
	// Must be a department in your account, and cannot be changed after creation.
	DepartmentID string `json:"department_id" validate:"required"`
	// Size of the labels printed at this station, given as width-by-height (for example, `1x1`).
	LabelSizeCode field.Optional[constants.LabelSizeCode] `json:"label_size,omitzero"`
	// Type of label printed at this station.
	//
	// - `tag`: a label attached to the physical product.
	// - `traveler`: a routing sheet that accompanies the batch through every production step.
	LabelTypeCode field.Optional[constants.LabelTypeCode] `json:"label_type,omitzero"`
}

var sampleLabelSizeCode = constants.LabelSizeCodeOneByOne
var sampleLabelTypeCode = constants.LabelTypeCodeTag
var sampleCreateScanningStationNotes = "Primary intake station on the receiving dock."

var sampleCreateScanningStationRequest = &CreateScanningStationRequest{
	Name:                apiresource.SampleScanningStationName,
	Notes:               field.Some(sampleCreateScanningStationNotes),
	Type:                constants.ScanningStationTypeInitBatch,
	OperatorRequirement: constants.OperatorRequirementNone,
	DepartmentID:        apiresource.SampleDepartmentID,
	LabelSizeCode:       field.Some(sampleLabelSizeCode),
	LabelTypeCode:       field.Some(sampleLabelTypeCode),
}

func (*CreateScanningStationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateScanningStationRequest)
}

// Creates a scanning station and assigns it to a department.
//
// The new station has no production steps connected to it; use Connect Production Steps to Scanning Station to attach them.
//
// Returns a conflict error if a scanning station with the same name already exists, and a not-found error if the department does not exist in your account.
type CreateScanningStationEndpoint struct{}

func (e *CreateScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateScanningStationRequest, *apiresource.ScanningStation] {
	return (&apiendpoint.APIEndpoint[*CreateScanningStationRequest, *apiresource.ScanningStation]{
		Title:               "Create Scanning Station",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/scanning-stations",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeScanningStation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
			return svc.(ScanningStationSvc).CreateScanningStation
		},
		LocationFunc: func(resp *apiresource.ScanningStation) string {
			return "/v1/operations/scanning-stations/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeScanningStation,
			Fields:     []string{"department", "production_steps"},
		}),
	})
}
