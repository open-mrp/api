package agentalertep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// AcknowledgeAlertRequest is a request to acknowledge an agent alert.
type AcknowledgeAlertRequest struct {
	// Alert ID.
	AlertID string `path:"id" validate:"required"`
}

// Marks an agent alert as acknowledged.
type AcknowledgeAlertEndpoint struct{}

func (e *AcknowledgeAlertEndpoint) Materialize() *apiendpoint.APIEndpoint[*AcknowledgeAlertRequest, *apiresource.AgentAlert] {
	return (&apiendpoint.APIEndpoint[*AcknowledgeAlertRequest, *apiresource.AgentAlert]{
		Title:             "Acknowledge Agent Alert",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/ai/alerts/{id}/actions/acknowledge",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AcknowledgeAlertRequest) (*apiresource.AgentAlert, *apierror.APIError) {
			return svc.(AgentAlertSvc).AcknowledgeAlert
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentAlert,
			Fields:     []string{"run", "action"},
		}),
	})
}
