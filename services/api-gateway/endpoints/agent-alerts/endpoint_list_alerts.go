package agentalertep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListAlertsRequest is a request to list agent alerts.
type ListAlertsRequest struct {
	apiresource.PaginationRequest
	// Filter by severity.
	Severity *constants.AgentAlertSeverity `query:"severity"`
	// Filter by alert status.
	Status *constants.AgentAlertStatus `query:"status"`
}

// Returns a paginated list of agent alerts for the current account.
type ListAlertsEndpoint struct{}

func (e *ListAlertsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAlertsRequest, *apiresource.List[apiresource.AgentAlert]] {
	return (&apiendpoint.APIEndpoint[*ListAlertsRequest, *apiresource.List[apiresource.AgentAlert]]{
		Title:             "List Agent Alerts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/ai/alerts",
		Request:           &ListAlertsRequest{},
		Response:          &apiresource.List[apiresource.AgentAlert]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAlertsRequest) (*apiresource.List[apiresource.AgentAlert], *apierror.APIError) {
			return svc.(AgentAlertSvc).ListAlerts
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAgentAlert,
			Fields:     []string{"run", "action"},
		}),
	}).WithDocSource(e)
}
