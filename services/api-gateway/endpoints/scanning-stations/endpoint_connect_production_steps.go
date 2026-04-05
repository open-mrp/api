package scanningstationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ConnectProductionStepsRequest is the request to connect production steps to a scanning station.
type ConnectProductionStepsRequest struct {
	// The ID of the scanning station.
	ScanningStationID string `path:"id" validate:"required"`
	// The name (or partial name) of production steps to connect.
	Name string `json:"name" validate:"required"`
}

var sampleConnectProductionStepsRequest = &ConnectProductionStepsRequest{
	Name: "Mixing",
}

func (*ConnectProductionStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleConnectProductionStepsRequest)
}

type ConnectProductionStepsEndpoint struct{}

func (e *ConnectProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ConnectProductionStepsRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*ConnectProductionStepsRequest, *apiresource.EmptyResource]{
		Title:             "Connect Production Steps to Scanning Station",
		Description:       "Connects production steps matching the provided name to a scanning station.",
		Method:            http.MethodPut,
		Route:             "/v1/operations/scanning-stations/{id}/production-steps",
		Request:           &ConnectProductionStepsRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ConnectProductionStepsRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ScanningStationSvc).ConnectProductionSteps
		},
	}
}
