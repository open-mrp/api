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

// TriggerRunRequest is the request to trigger a new agent run.
type TriggerRunRequest struct {
	// The ID of the agent definition to run.
	AgentDefinitionID string `json:"agent_definition_id" validate:"required"`
	// Optional input text to provide to the agent at the start of the run.
	Input string `json:"input,omitempty"`
}

var sampleTriggerRunRequest = &TriggerRunRequest{
	AgentDefinitionID: apiresource.SampleAgentDefinitionID,
	Input:             "Process the latest incoming orders.",
}

func (*TriggerRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleTriggerRunRequest)
}

type TriggerRunEndpoint struct{}

func (e *TriggerRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*TriggerRunRequest, *apiresource.AgentRun] {
	return &apiendpoint.APIEndpoint[*TriggerRunRequest, *apiresource.AgentRun]{
		Title:             "Trigger Run",
		Description:       "Triggers a new agent run for the specified agent definition.",
		Method:            http.MethodPost,
		Route:             "/v1/ai/runs",
		ContentType:       "application/json",
		Request:           &TriggerRunRequest{},
		Response:          &apiresource.AgentRun{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *TriggerRunRequest) (*apiresource.AgentRun, *apierror.APIError) {
			return svc.(AgentRunSvc).TriggerAgentRun
		},
		LocationFunc: func(resp *apiresource.AgentRun) string {
			return "/v1/ai/runs/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentRun,
			Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
		}),
	}
}
