package scanningstationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list scanning stations.
type ListScanningStationsRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return stations at one of these work centers.
	WorkCenterIDs []string `query:"work_center_ids" validate:"omitempty,max=100"`
}

// Returns a list of your scanning stations, newest first.
type ListScanningStationsEndpoint struct{}

func (e *ListScanningStationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListScanningStationsRequest, *apiresource.List[apiresource.ScanningStation]] {
	return (&apiendpoint.APIEndpoint[*ListScanningStationsRequest, *apiresource.List[apiresource.ScanningStation]]{
		Title:             "List Scanning Stations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeScanningStation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListScanningStationsRequest) (*apiresource.List[apiresource.ScanningStation], *apierror.APIError) {
			return svc.(ScanningStationSvc).ListScanningStations
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeScanningStation,
			Fields:     []string{"work_center", "production_steps"},
		}),
	})
}
