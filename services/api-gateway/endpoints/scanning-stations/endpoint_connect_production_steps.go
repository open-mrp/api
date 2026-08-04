package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to connect production steps to a scanning station.
type ConnectProductionStepsRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
	// Full or partial production step name to match.
	//
	// Matching is a case-insensitive substring match, so a broad value such as a single letter can capture far more steps than intended.
	Name string `json:"name" validate:"required"`
}

var sampleConnectProductionStepsRequest = &ConnectProductionStepsRequest{
	Name: "Mixing",
}

func (*ConnectProductionStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleConnectProductionStepsRequest)
}

// Connects production steps to a scanning station by name.
//
// Every production step in your account whose name contains the provided value is connected. A production step can be connected to at most one scanning station, so matching steps are moved off any station they were previously connected to. Steps already connected to this station that do not match are left connected, so this adds to the station's steps rather than replacing them.
//
// Nothing about the station is returned, so retrieve the scanning station afterward to confirm which steps are now connected.
type ConnectProductionStepsEndpoint struct{}

func (e *ConnectProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ConnectProductionStepsRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ConnectProductionStepsRequest, *apiresource.EmptyResource]{
		Title:               "Connect Production Steps to Scanning Station",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/operations/scanning-stations/{id}/production-steps",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ConnectProductionStepsRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ScanningStationSvc).ConnectProductionSteps
		},
	})
}
