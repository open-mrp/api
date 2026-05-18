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

// Request to continue an agent run awaiting input.
type ContinueRunRequest struct {
	// Agent run ID.
	AgentRunID string `path:"id" validate:"required"`
	// User message to send to the agent.
	Message string `json:"message" validate:"required"`
	// Tool slugs to approve individually. If empty, all pending tools are approved.
	ApprovedToolSlugs []string `json:"approved_tool_slugs"`
	// Tool slugs to allow for the rest of the run without further approval.
	AllowedToolSlugs []string `json:"allowed_tool_slugs"`
}

var sampleContinueRunRequest = &ContinueRunRequest{
	Message: "Yes, proceed with creating the order.",
}

func (*ContinueRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleContinueRunRequest)
}

// Continues an agent run awaiting input with a user message.
type ContinueRunEndpoint struct{}

func (e *ContinueRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*ContinueRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*ContinueRunRequest, *apiresource.AgentRun]{
		Title:             "Continue Run",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/ai/runs/{id}/actions/continue",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ContinueRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).ContinueAgentRun
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
		}),
	})
}
