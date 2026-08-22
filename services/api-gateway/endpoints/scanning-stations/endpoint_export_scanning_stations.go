package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Filters which scanning stations land in the exported file.
type ExportScanningStationsRequest struct {
	// Free-text search term matched against scanning station names.
	Query *string `json:"q"`
}

var sampleExportScanningStationsRequest = &ExportScanningStationsRequest{}

func (*ExportScanningStationsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportScanningStationsRequest)
}

// Starts an export of every matching scanning station and returns the job that tracks it.
type ExportScanningStationsEndpoint struct{}

func (e *ExportScanningStationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportScanningStationsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportScanningStationsRequest, *apiresource.Job]{
		Title:             "Export Scanning Stations",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportScanningStationsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ScanningStationSvc).ExportScanningStations
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
