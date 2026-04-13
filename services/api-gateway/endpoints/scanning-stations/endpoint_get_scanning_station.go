package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a scanning station.
type GetScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
}

type GetScanningStationEndpoint struct{}

func (e *GetScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetScanningStationRequest, *apiresource.ScanningStation] {
	return &apiendpoint.APIEndpoint[*GetScanningStationRequest, *apiresource.ScanningStation]{
		Title:             "Get Scanning Station",
		Description:       "Returns a scanning station by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}",
		Request:           &GetScanningStationRequest{},
		Response:          &apiresource.ScanningStation{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
			return svc.(ScanningStationSvc).GetScanningStation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType:    constants.ObjectTypeScanningStation,
			Fields:        []string{"department", "production_steps"},
			DefaultFields: []string{"department"},
		}),
	}
}
