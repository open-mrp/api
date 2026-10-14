package agentrunep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list agent runs.
type ListRunsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Restricts results to runs in this status.
	Status *constants.AgentRunStatus `query:"status"`
	// Restricts results to runs of a single agent.
	AgentDefinitionID *string `query:"agent_definition_id"`
}

// Lists agent runs for your account, newest first.
//
// The `q` parameter matches a run's ID, its status, or the ID of the agent that produced it.
type ListRunsEndpoint struct{}

func (e *ListRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRunsRequest, *apiresource.List[apiresource.AgentRun]] {
	return (&apiendpoint.APIEndpoint[*ListRunsRequest, *apiresource.List[apiresource.AgentRun]]{
		Title:             "List Agent Runs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/runs",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentRun,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAgentRuns, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRunsRequest) (*apiresource.List[apiresource.AgentRun], *apierror.APIError) {
			return svc.(AgentRunSvc).ListRuns
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"triggered_by", "definition", "actions", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
