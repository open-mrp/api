package scanningstationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a scanning station.
type RetrieveScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
}

// Returns a scanning station by ID.
type RetrieveScanningStationEndpoint struct{}

func (e *RetrieveScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveScanningStationRequest, *apiresource.ScanningStation] {
	return (&apiendpoint.APIEndpoint[*RetrieveScanningStationRequest, *apiresource.ScanningStation]{
		Title:             "Retrieve Scanning Station",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeScanningStation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
			return svc.(ScanningStationSvc).GetScanningStation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeScanningStation,
			Fields:     []string{"work_center", "production_steps"},
		}),
	})
}
