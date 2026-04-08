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

// UpdateScanningStationRequest is the request to partially update a scanning station.
type UpdateScanningStationRequest struct {
	// The ID of the scanning station to update.
	ScanningStationID string `path:"id" validate:"required"`
	// The display name of the scanning station.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Optional notes about the scanning station.
	Notes *string `json:"notes,omitempty"`
	// The label size code for the scanning station.
	LabelSizeCode *constants.LabelSizeCode `json:"label_size_code,omitempty" nullable:"false"`
	// The label type code for the scanning station.
	LabelTypeCode *constants.LabelTypeCode `json:"label_type_code,omitempty" nullable:"false"`
	// Whether material check is required at this station.
	MaterialCheckRequired *bool `json:"material_check_required,omitempty"`
}

var sampleUpdateScanningStationName = "Station B"
var sampleUpdateScanningStationRequest = &UpdateScanningStationRequest{
	Name: &sampleUpdateScanningStationName,
}

func (*UpdateScanningStationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateScanningStationRequest)
}

type UpdateScanningStationEndpoint struct{}

func (e *UpdateScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateScanningStationRequest, *apiresource.ScanningStation] {
	return &apiendpoint.APIEndpoint[*UpdateScanningStationRequest, *apiresource.ScanningStation]{
		Title:             "Update Scanning Station",
		Description:       "Partially updates a scanning station.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/scanning-stations/{id}",
		Request:           &UpdateScanningStationRequest{},
		Response:          &apiresource.ScanningStation{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
			return svc.(ScanningStationSvc).UpdateScanningStation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType:    constants.ObjectTypeScanningStation,
			Fields:        []string{"department", "production_steps"},
			DefaultFields: []string{"department"},
		}),
	}
}
