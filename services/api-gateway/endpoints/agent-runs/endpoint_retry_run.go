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

// Request to retry a failed agent run.
type RetryRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
}

// Retries a failed agent run by resuming its existing transcript.
//
// Only runs in the `failed` status can be retried; retrying a run in any other status returns a validation error. The run is re-attempted from where it left off — its prior reasoning and tool results are replayed, so the agent continues with full knowledge of what it already did rather than starting over, which minimizes the chance of it repeating side effects it has already caused.
//
// A run can be retried at most five times in total, and any automatic retries the platform already performed for transient failures count against that budget.
type RetryRunEndpoint struct{}

func (e *RetryRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetryRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*RetryRunRequest, *apiresource.AgentRun]{
		Title:               "Retry Agent Run",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/ai/runs/{id}/actions/retry",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentRun,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentRuns, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetryRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).RetryAgentRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
