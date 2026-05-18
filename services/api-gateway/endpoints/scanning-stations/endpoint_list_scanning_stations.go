package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list scanning stations.
type ListScanningStationsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of scanning stations for the current account.
type ListScanningStationsEndpoint struct{}

func (e *ListScanningStationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListScanningStationsRequest, *apiresource.List[apiresource.ScanningStation]] {
	return (&apiendpoint.APIEndpoint[*ListScanningStationsRequest, *apiresource.List[apiresource.ScanningStation]]{
		Title:             "List Scanning Stations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations",
		Request:           &ListScanningStationsRequest{},
		Response:          &apiresource.List[apiresource.ScanningStation]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListScanningStationsRequest) (*apiresource.List[apiresource.ScanningStation], *apierror.APIError) {
			return svc.(ScanningStationSvc).ListScanningStations
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeScanningStation,
			Fields:     []string{"department", "production_steps"},
		}),
	}).WithDocSource(e)
}
