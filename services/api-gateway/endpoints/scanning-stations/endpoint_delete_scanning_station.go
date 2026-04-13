package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a scanning station.
type DeleteScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
}

type DeleteScanningStationEndpoint struct{}

func (e *DeleteScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteScanningStationRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteScanningStationRequest, *apiresource.EmptyResource]{
		Title:             "Delete Scanning Station",
		Description:       "Deletes a scanning station.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}",
		Request:           &DeleteScanningStationRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteScanningStationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ScanningStationSvc).DeleteScanningStation
		},
	}
}
