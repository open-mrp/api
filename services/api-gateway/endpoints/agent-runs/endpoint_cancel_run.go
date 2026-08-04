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

// Request to cancel an agent run.
type CancelRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
}

// Cancels an in-progress agent run.
//
// A run can be cancelled while it is working or paused waiting on the user — `pending`, `running`, `awaiting_input`, or `awaiting_approval`. Cancelling a run in a terminal status (`completed`, `failed`, `cancelled`) returns a validation error.
//
// Cancelling a run that is `awaiting_approval` counts as denying the review: every action still pending review is recorded as rejected, attributed to the caller. Work the agent already completed is not undone.
type CancelRunEndpoint struct{}

func (e *CancelRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*CancelRunRequest, *apiresource.AgentRun]{
		Title:               "Cancel Agent Run",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/ai/runs/{id}/actions/cancel",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentRun,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentRuns, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CancelRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).CancelAgentRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
