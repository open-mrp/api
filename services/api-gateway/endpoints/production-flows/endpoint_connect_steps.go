package productionflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ConnectStepsRequest is the request to connect two steps in the production flow DAG.
type ConnectStepsRequest struct {
	// Source (upstream) production step ID.
	SourceProductionStepID string `json:"source_production_step_id" validate:"required"`
	// Target (downstream) production step ID.
	TargetProductionStepID string `json:"target_production_step_id" validate:"required"`
}

var sampleConnectStepsRequest = &ConnectStepsRequest{
	SourceProductionStepID: apiresource.SampleProductionStepID,
	TargetProductionStepID: apiresource.SampleProductionStepID,
}

func (*ConnectStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleConnectStepsRequest)
}

type ConnectStepsEndpoint struct{}

func (e *ConnectStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ConnectStepsRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*ConnectStepsRequest, *apiresource.EmptyResource]{
		Title:             "Connect Production Steps",
		Description:       "Connects two production steps in the production flow DAG. The source step becomes an upstream dependency of the target step.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/production-flows/actions/connect-steps",
		ContentType:       "application/json",
		Request:           &ConnectStepsRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ConnectStepsRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionFlowSvc).ConnectSteps
		},
	}
}
