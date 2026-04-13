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

// CreateScanningStationRequest is the request to create a new scanning station.
type CreateScanningStationRequest struct {
	// The display name of the scanning station.
	Name string `json:"name" validate:"required,max=255"`
	// Optional notes about the scanning station.
	Notes *string `json:"notes,omitempty" nullable:"false"`
	// The type of scanning station.
	Type constants.ScanningStationType `json:"type" validate:"required"`
	// Whether material check is required at this station.
	MaterialCheckRequired bool `json:"material_check_required"`
	// The ID of the department to associate with this scanning station.
	DepartmentID string `json:"department_id" validate:"required,max=191"`
	// The label size code for the scanning station.
	LabelSizeCode *constants.LabelSizeCode `json:"label_size,omitempty" nullable:"false"`
	// The label type code for the scanning station.
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
		Description:       "Creates a new scanning station associated with a department.",
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
