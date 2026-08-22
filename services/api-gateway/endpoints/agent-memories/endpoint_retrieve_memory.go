package agentmemoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an agent memory.
type RetrieveMemoryRequest struct {
	// Memory ID.
	ID string `path:"id" validate:"required"`
}

// Returns an agent memory by ID.
//
// An expired memory is still returned here, even though it is excluded from list results and no longer recalled by agents.
type RetrieveMemoryEndpoint struct{}

func (e *RetrieveMemoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMemoryRequest, *apiresource.AgentMemory] {
	return (&apiendpoint.APIEndpoint[*RetrieveMemoryRequest, *apiresource.AgentMemory]{
		Title:               "Retrieve Agent Memory",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/ai/memories/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAgentMemory,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAgentMemories, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
			return svc.(AgentMemorySvc).GetMemory
		},
	})
}
