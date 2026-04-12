package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListRunsRequest is the request to list agent runs.
type ListRunsRequest struct {
	apiresource.PaginationRequest
	// Filter by run status code (e.g. "running", "completed", "failed").
	StatusCode *string `query:"status"`
	// Filter by agent definition ID.
	AgentDefinitionID *string `query:"agent_definition_id"`
}

type ListRunsEndpoint struct{}

func (e *ListRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRunsRequest, *apiresource.List[apiresource.AgentRun]] {
	return &apiendpoint.APIEndpoint[*ListRunsRequest, *apiresource.List[apiresource.AgentRun]]{
		Title:             "List Runs",
		Description:       "Returns a paginated list of agent runs for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/runs",
		Request:           &ListRunsRequest{},
		Response:          &apiresource.List[apiresource.AgentRun]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRunsRequest) (*apiresource.List[apiresource.AgentRun], *apierror.APIError) {
			return svc.(AgentRunSvc).ListRuns
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"definition", "actions", "definition.config", "definition.tools", "definition.role"},
		}),
	}
}
