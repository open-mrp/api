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
	"github.com/augno/api/shared/field"
)

// Request to trigger an agent run.
type TriggerRunRequest struct {
	// ID of the agent definition to run.
	//
	// The agent must be active for the account; triggering an inactive agent returns a validation error.
	AgentDefinitionID string `json:"agent_definition_id" validate:"required"`
	// Instruction text passed to the agent at the start of the run.
	//
	// Recorded on the run as `{"message": <input>}` in its `input` field.
	Input field.Optional[string] `json:"input,omitzero"`
}

var sampleTriggerRunRequest = &TriggerRunRequest{
	AgentDefinitionID: apiresource.SampleAgentDefinitionID,
	Input:             field.Some("Process the latest incoming orders."),
}

func (*TriggerRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleTriggerRunRequest)
}

// Starts a new run of the specified agent.
//
// The run is created in the `pending` status and executed asynchronously; poll Retrieve Agent Run to follow its progress.
type TriggerRunEndpoint struct{}

func (e *TriggerRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*TriggerRunRequest, *apiresource.AgentRun] {
	return (&apiendpoint.APIEndpoint[*TriggerRunRequest, *apiresource.AgentRun]{
		Title:               "Trigger Agent Run",
		Method:              http.MethodPost,
		Route:               "/v1/ai/runs",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentRun,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentRuns, Action: types.ActionCreate}},
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
