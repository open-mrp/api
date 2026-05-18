package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an agent run.
type RetrieveRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
}

// Returns an agent run by ID.
type RetrieveRunEndpoint struct{}

func (e *RetrieveRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*RetrieveRunRequest, *apiresource.AgentRun]{
		Title:             "Retrieve Run",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/runs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).GetRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "steps", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
