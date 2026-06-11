package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to resume a paused agent run.
type ContinueRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
	// User message to send to the agent.
	Message string `json:"message" validate:"required"`
	// Slugs of tools whose pending calls should be approved.
	//
	// When empty, all pending tool calls are approved. Approvals are one-time: later calls to the same tool pause for review again unless the slug is also in `allowed_tool_slugs`.
	ApprovedToolSlugs []string `json:"approved_tool_slugs"`
	// Slugs of tools to allow for the rest of the run.
	//
	// Allowed tools execute without pausing for review; slugs accumulate across continue requests for the life of the run.
	AllowedToolSlugs []string `json:"allowed_tool_slugs"`
}

var sampleContinueRunRequest = &ContinueRunRequest{
	Message: "Yes, proceed with creating the order.",
}

func (*ContinueRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleContinueRunRequest)
}

// Resumes a paused agent run with a user message and any tool approvals.
//
// The run must be in the `awaiting_input` or `awaiting_approval` status.
type ContinueRunEndpoint struct{}

func (e *ContinueRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*ContinueRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*ContinueRunRequest, *apiresource.AgentRun]{
		Title:             "Continue Agent Run",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/ai/runs/{id}/actions/continue",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentRun,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ContinueRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).ContinueAgentRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
