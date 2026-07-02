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

// Request to retrieve an agent run.
type RetrieveRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
}

// Returns an agent run by ID.
type RetrieveRunEndpoint struct{}

func (e *RetrieveRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*RetrieveRunRequest, *apiresource.AgentRun]{
		Title:               "Retrieve Agent Run",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/ai/runs/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentRun,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentRuns, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).GetRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"triggered_by", "actions", "definition", "steps", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
