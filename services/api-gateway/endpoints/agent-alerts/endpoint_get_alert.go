package agentalertep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetAlertRequest is a request to get an agent alert.
type GetAlertRequest struct {
	// Alert ID.
	AlertID string `path:"id" validate:"required"`
}

type GetAlertEndpoint struct{}

func (e *GetAlertEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAlertRequest, *apiresource.AgentAlert] {
	return &apiendpoint.APIEndpoint[*GetAlertRequest, *apiresource.AgentAlert]{
		Title:             "Get Agent Alert",
		Description:       "Returns an agent alert by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/alerts/{id}",
		Request:           &GetAlertRequest{},
		Response:          &apiresource.AgentAlert{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAlertRequest) (*apiresource.AgentAlert, *apierror.APIError) {
			return svc.(AgentAlertSvc).GetAlert
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentAlert,
			Fields:     []string{"run", "action"},
		}),
	}
}
