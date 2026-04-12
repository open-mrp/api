package priorityep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetPriorityRequest is the request to retrieve a single priority.
type GetPriorityRequest struct {
	// The ID or code of the priority to retrieve.
	PriorityID string `path:"id" validate:"required"`
}

type GetPriorityEndpoint struct{}

func (e *GetPriorityEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPriorityRequest, *apiresource.Priority] {
	return &apiendpoint.APIEndpoint[*GetPriorityRequest, *apiresource.Priority]{
		Title:             "Get Priority",
		Description:       "Returns a single priority by its ID or code.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/priorities/{id}",
		Request:           &GetPriorityRequest{},
		Response:          &apiresource.Priority{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPriorityRequest) (*apiresource.Priority, *apierror.APIError) {
			return svc.(PrioritySvc).GetPriority
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePriority,
			Fields:     []string{"owner"},
		}),
	}
}
