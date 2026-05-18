package priorityep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a priority.
type RetrievePriorityRequest struct {
	// Priority ID or code.
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
		Request:           &RetrievePriorityRequest{},
		Response:          &apiresource.Priority{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePriorityRequest) (*apiresource.Priority, *apierror.APIError) {
			return svc.(PrioritySvc).GetPriority
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePriority,
			Fields:     []string{"owner"},
		}),
	}).WithDocSource(e)
}
