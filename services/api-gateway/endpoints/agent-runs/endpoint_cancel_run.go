package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to cancel an agent run.
type CancelRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
}

// Cancels a running or pending agent run.
type CancelRunEndpoint struct{}

func (e *CancelRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*CancelRunRequest, *apiresource.AgentRun]{
		Title:             "Cancel Run",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/ai/runs/{id}/actions/cancel",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentRun,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CancelRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).CancelAgentRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
