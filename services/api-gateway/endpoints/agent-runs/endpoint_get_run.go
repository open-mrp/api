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
type GetRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
}

type GetRunEndpoint struct{}

func (e *GetRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetRunRequest, *apiresource.AgentRun] {
	return &apiendpoint.APIEndpoint[*GetRunRequest, *apiresource.AgentRun]{
		Title:             "Get Run",
		Description:       "Returns an agent run by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/runs/{id}",
		Request:           &GetRunRequest{},
		Response:          &apiresource.AgentRun{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).GetRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "steps", "definition.config", "definition.tools", "definition.role"},
		}),
	}
}
