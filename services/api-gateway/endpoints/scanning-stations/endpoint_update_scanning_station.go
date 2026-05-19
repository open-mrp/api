package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
)

// Request to partially update a scanning station.
type UpdateScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Notes.
	Notes *patch.Field[string] `json:"notes,omitempty"`
	// Label size code.
	LabelSizeCode *constants.LabelSizeCode `json:"label_size,omitempty"`
	// Label type code.
	LabelTypeCode *constants.LabelTypeCode `json:"label_type,omitempty"`
	// Operator requirement behavior for this station.
	OperatorRequirement *constants.OperatorRequirement `json:"operator_requirement,omitempty"`
}

var sampleUpdateScanningStationName = "Station B"
var sampleUpdateScanningStationRequest = &UpdateScanningStationRequest{
	Name: &sampleUpdateScanningStationName,
}

func (*UpdateScanningStationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateScanningStationRequest)
}

// Partially updates a scanning station.
type UpdateScanningStationEndpoint struct{}

func (e *UpdateScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateScanningStationRequest, *apiresource.ScanningStation] {
	return (&apiendpoint.APIEndpoint[*UpdateScanningStationRequest, *apiresource.ScanningStation]{
		Title:             "Update Scanning Station",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
			return svc.(ScanningStationSvc).UpdateScanningStation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeScanningStation,
			Fields:     []string{"department", "production_steps"},
		}),
	})
}
