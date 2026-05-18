package agentalertep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveAlertRequest is a request to get an agent alert.
type RetrieveAlertRequest struct {
	// Alert ID.
	AlertID string `path:"id" validate:"required"`
}

// Returns an agent alert by ID.
type RetrieveAlertEndpoint struct{}

func (e *RetrieveAlertEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAlertRequest, *apiresource.AgentAlert] {
	return (&apiendpoint.APIEndpoint[*RetrieveAlertRequest, *apiresource.AgentAlert]{
		Title:             "Retrieve Agent Alert",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/alerts/{id}",
		Request:           &RetrieveAlertRequest{},
		Response:          &apiresource.AgentAlert{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAlertRequest) (*apiresource.AgentAlert, *apierror.APIError) {
			return svc.(AgentAlertSvc).GetAlert
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentAlert,
			Fields:     []string{"run", "action"},
		}),
	}).WithDocSource(e)
}
