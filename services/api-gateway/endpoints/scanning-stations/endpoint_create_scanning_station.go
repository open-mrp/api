package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a scanning station.
type CreateScanningStationRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Notes.
	Notes *string `json:"notes,omitempty" nullable:"false"`
	// Scanning station type.
	Type constants.ScanningStationType `json:"type" validate:"required"`
	// Operator requirement behavior for this station.
	OperatorRequirement constants.OperatorRequirement `json:"operator_requirement" validate:"required"`
	// Department ID.
	DepartmentID string `json:"department_id" validate:"required"`
	// Label size code.
	LabelSizeCode *constants.LabelSizeCode `json:"label_size,omitempty" nullable:"false"`
	// Label type code.
	LabelTypeCode *constants.LabelTypeCode `json:"label_type,omitempty" nullable:"false"`
}

var sampleLabelSizeCode = constants.LabelSizeCodeOneByOne
var sampleLabelTypeCode = constants.LabelTypeCodeTag

var sampleCreateScanningStationRequest = &CreateScanningStationRequest{
	Name:                apiresource.SampleScanningStationName,
	Type:                constants.ScanningStationTypeInitBatch,
	OperatorRequirement: constants.OperatorRequirementNone,
	DepartmentID:        apiresource.SampleDepartmentID,
	LabelSizeCode:       &sampleLabelSizeCode,
	LabelTypeCode:       &sampleLabelTypeCode,
}

func (*CreateScanningStationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateScanningStationRequest)
}

// Creates a scanning station associated with a department.
type CreateScanningStationEndpoint struct{}

func (e *CreateScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateScanningStationRequest, *apiresource.ScanningStation] {
	return (&apiendpoint.APIEndpoint[*CreateScanningStationRequest, *apiresource.ScanningStation]{
		Title:             "Create Scanning Station",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations",
		Request:           &CreateScanningStationRequest{},
		Response:          &apiresource.ScanningStation{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
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
	}).WithDocSource(e)
}
