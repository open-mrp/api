package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list agent runs.
type ListRunsRequest struct {
	apiresource.PaginationRequest
	// Filter to runs with this status (e.g. `running`, `completed`, `failed`).
	StatusCode *string `query:"status"`
	// Filter to runs of a specific agent definition.
	AgentDefinitionID *string `query:"agent_definition_id"`
}

// Returns a paginated list of agent runs for the current account.
type ListRunsEndpoint struct{}

func (e *ListRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRunsRequest, *apiresource.List[apiresource.AgentRun]] {
	return (&apiendpoint.APIEndpoint[*ListRunsRequest, *apiresource.List[apiresource.AgentRun]]{
		Title:               "List Agent Runs",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/ai/runs",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentRun,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentRuns, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRunsRequest) (*apiresource.List[apiresource.AgentRun], *apierror.APIError) {
			return svc.(AgentRunSvc).ListRuns
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"triggered_by", "definition", "actions", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
