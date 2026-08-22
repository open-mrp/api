package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list agent runs.
type ListRunsRequest struct {
	apiresource.PaginationRequest
	// Restricts results to runs in this status.
	//
	// One of `pending`, `running`, `awaiting_input`, `awaiting_approval`, `completed`, `failed`, or `cancelled`.
	StatusCode *string `query:"status"`
	// Restricts results to runs of a single agent.
	AgentDefinitionID *string `query:"agent_definition_id"`
}

// Lists agent runs for your account, newest first.
//
// The `q` parameter matches a run's ID, its status, or the ID of the agent that produced it.
type ListRunsEndpoint struct{}

func (e *ListRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRunsRequest, *apiresource.List[apiresource.AgentRun]] {
	return (&apiendpoint.APIEndpoint[*ListRunsRequest, *apiresource.List[apiresource.AgentRun]]{
		Title:               "List Agent Runs",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/ai/runs",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
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
