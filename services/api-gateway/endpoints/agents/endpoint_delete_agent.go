package agentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a custom agent definition.
type DeleteAgentRequest struct {
	// Agent definition ID.
	AgentDefinitionID string `path:"id" validate:"required"`
}

type DeleteAgentEndpoint struct{}

func (e *DeleteAgentEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAgentRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteAgentRequest, *apiresource.EmptyResource]{
		Title:             "Delete Agent",
		Description:       "Soft-deletes a custom agent definition. System agents cannot be deleted.",
		Method:            http.MethodDelete,
		Route:             "/v1/ai/agents/{id}",
		ContentType:       "application/json",
		Request:           &DeleteAgentRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAgentRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AgentSvc).DeleteAgent
		},
	}
}
