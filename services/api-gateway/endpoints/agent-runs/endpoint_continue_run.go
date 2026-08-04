package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to resume a paused agent run.
type ContinueRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
	// Message to send to the agent as the next turn of the run.
	//
	// It accompanies any approval or denial in the same request, so use it to tell the agent how to proceed with what you just allowed or blocked.
	Message string `json:"message" validate:"required"`
	// Slugs of tools whose pending calls should be approved.
	//
	// Approves every call currently pending review for each named tool. Approval is one-time — the next call to the same tool pauses for review again. Tools you do not name are left pending, and the run resumes without them.
	ApprovedToolSlugs []string `json:"approved_tool_slugs,omitzero"`
	// Slugs of tools whose pending calls should be denied.
	//
	// The run keeps going: each denied call is answered with a "denied by user" result so the agent proceeds without it, instead of cancelling the run. A single resume may both approve and reject different tools.
	RejectedToolSlugs []string `json:"rejected_tool_slugs,omitzero"`
	// Tool-call IDs (the `tool_use_id` of individual blocked calls) to approve.
	//
	// Use this instead of `approved_tool_slugs` to approve ONE specific call when several pending calls share the same tool slug — approving by slug would approve all of them. Approvals are one-time.
	ApprovedToolCallIDs []string `json:"approved_tool_call_ids,omitzero"`
	// Tool-call IDs (the `tool_use_id` of individual blocked calls) to deny.
	//
	// Per-call counterpart of `rejected_tool_slugs`, letting you deny one specific call among several that share a slug. Each denied call is answered with a "denied by user" result and the run continues.
	RejectedToolCallIDs []string `json:"rejected_tool_call_ids,omitzero"`
}

var sampleContinueRunRequest = &ContinueRunRequest{
	Message: "Yes, proceed with creating the order.",
}

func (*ContinueRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleContinueRunRequest)
}

// Resumes a paused agent run with a user message and any tool review decisions.
//
// The run must be `awaiting_input` or `awaiting_approval`; resuming it from any other status returns a validation error. It moves back to `running` and continues asynchronously, so poll Retrieve Agent Run to follow it. Each approval and denial is recorded on the matching action and attributed to the caller.
type ContinueRunEndpoint struct{}

func (e *ContinueRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*ContinueRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*ContinueRunRequest, *apiresource.AgentRun]{
		Title:               "Continue Agent Run",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/ai/runs/{id}/actions/continue",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentRun,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentRuns, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ContinueRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).ContinueAgentRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
