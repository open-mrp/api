package priorityep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a priority.
type RetrievePriorityRequest struct {
	// Priority ID or code.
	//
	// Passing the code, such as `normal`, resolves the same priority as passing its generated ID.
	PriorityID string `path:"id" validate:"required"`
}

// Retrieves a single priority level by ID or by code.
//
// Looking one up by code is usually more convenient, because other resources refer to a priority by code rather than by ID.
type RetrievePriorityEndpoint struct{}

func (e *RetrievePriorityEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePriorityRequest, *apiresource.Priority] {
	return (&apiendpoint.APIEndpoint[*RetrievePriorityRequest, *apiresource.Priority]{
		Title:             "Retrieve Priority",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/priorities/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPriorities, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypePriority,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePriorityRequest) (*apiresource.Priority, *apierror.APIError) {
			return svc.(PrioritySvc).GetPriority
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePriority,
			Fields:     []string{"owner"},
		}),
	})
}
