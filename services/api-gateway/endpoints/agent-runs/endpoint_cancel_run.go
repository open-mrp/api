package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// CancelRunRequest is the request to cancel an agent run.
type CancelRunRequest struct {
	// The ID of the agent run to cancel.
	AgentRunID string `path:"id" validate:"required"`
}

type CancelRunEndpoint struct{}

func (e *CancelRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelRunRequest, *apiresource.AgentRun] {
	return &apiendpoint.APIEndpoint[*CancelRunRequest, *apiresource.AgentRun]{
		Title:             "Cancel Run",
		Description:       "Cancels a running or pending agent run.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/ai/runs/{id}/actions/cancel",
		Request:           &CancelRunRequest{},
		Response:          &apiresource.AgentRun{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CancelRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).CancelAgentRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
		}),
	}
}
