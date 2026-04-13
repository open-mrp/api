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
	// Whether material check is required.
	MaterialCheckRequired bool `json:"material_check_required"`
	// Department ID.
	DepartmentID string `json:"department_id" validate:"required,max=191"`
	// Label size code.
	LabelSizeCode *constants.LabelSizeCode `json:"label_size,omitempty" nullable:"false"`
	// Label type code.
	LabelTypeCode *constants.LabelTypeCode `json:"label_type,omitempty" nullable:"false"`
}

var sampleCreateScanningStationRequest = &CreateScanningStationRequest{
	Name:                  apiresource.SampleScanningStationName,
	Type:                  constants.ScanningStationTypeInitBatch,
	MaterialCheckRequired: false,
	DepartmentID:          apiresource.SampleDepartmentID,
}

func (*CreateScanningStationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateScanningStationRequest)
}

type CreateScanningStationEndpoint struct{}

func (e *CreateScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateScanningStationRequest, *apiresource.ScanningStation] {
	return &apiendpoint.APIEndpoint[*CreateScanningStationRequest, *apiresource.ScanningStation]{
		Title:             "Create Scanning Station",
		Description:       "Creates a scanning station associated with a department.",
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
			ObjectType:    constants.ObjectTypeScanningStation,
			Fields:        []string{"department", "production_steps"},
			DefaultFields: []string{"department"},
		}),
	}
}
