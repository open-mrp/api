package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a scanning station.
type DeleteScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
}

// Deletes a scanning station.
//
// Production steps connected to the station are not deleted, but they are left without a station to scan at until you connect them to another one. Deleting a station that was already deleted returns an already-deleted error rather than a not-found error.
type DeleteScanningStationEndpoint struct{}

func (e *DeleteScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteScanningStationRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteScanningStationRequest, *apiresource.EmptyResource]{
		Title:               "Delete Scanning Station",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/operations/scanning-stations/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteScanningStationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ScanningStationSvc).DeleteScanningStation
		},
	})
}
