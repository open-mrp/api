package priorityep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a priority.
type RetrievePriorityRequest struct {
	// Priority ID or code.
	//
	// Accepts either a priority ID or one of the codes `low`, `normal`, `high`.
	PriorityID string `path:"id" validate:"required"`
}

// Returns a priority by ID or code.
type RetrievePriorityEndpoint struct{}

func (e *RetrievePriorityEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePriorityRequest, *apiresource.Priority] {
	return (&apiendpoint.APIEndpoint[*RetrievePriorityRequest, *apiresource.Priority]{
		Title:             "Retrieve Priority",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/priorities/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
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
