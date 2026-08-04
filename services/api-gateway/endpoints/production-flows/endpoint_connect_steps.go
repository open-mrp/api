package productionflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to connect two production steps.
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

// Connects two production steps so that work flows from the source step into the target step.
//
// The source step becomes an upstream dependency of the target step, and connecting a pair that is already connected has no effect.
//
// Connections are otherwise derived from item relationships: changing which items a step produces or consumes recomputes every connection on that step, which discards connections made here.
type ConnectStepsEndpoint struct{}

func (e *ConnectStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ConnectStepsRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ConnectStepsRequest, *apiresource.EmptyResource]{
		Title:             "Connect Production Steps",
		Method:            http.MethodPost,
		Route:             "/v1/operations/production-flows/actions/connect-steps",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSteps, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ConnectStepsRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionFlowSvc).ConnectSteps
		},
	})
}
