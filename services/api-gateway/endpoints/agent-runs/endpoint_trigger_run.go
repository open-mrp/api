package agentrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to trigger an agent run.
type TriggerRunRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `json:"agent_definition_id" validate:"required"`
	// Input text for the agent.
	Input field.Optional[string] `json:"input,omitzero"`
}

var sampleTriggerRunRequest = &TriggerRunRequest{
	AgentDefinitionID: apiresource.SampleAgentDefinitionID,
	Input:             field.Some("Process the latest incoming orders."),
}

func (*TriggerRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleTriggerRunRequest)
}

// Triggers an agent run for the specified agent definition.
type TriggerRunEndpoint struct{}

func (e *TriggerRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*TriggerRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*TriggerRunRequest, *apiresource.AgentRun]{
		Title:             "Trigger Run",
		Method:            http.MethodPost,
		Route:             "/v1/ai/runs",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAgentRun,
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
	})
}
