package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
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
		Title:               "Retrieve Scanning Station",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/scanning-stations/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeScanningStation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
			return svc.(ScanningStationSvc).GetScanningStation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeScanningStation,
			Fields:     []string{"department", "production_steps"},
		}),
	})
}
