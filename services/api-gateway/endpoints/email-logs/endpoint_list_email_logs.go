package emaillogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list email logs.
type ListEmailLogsRequest struct {
	apiresource.PaginationRequest
}

type ListEmailLogsEndpoint struct{}

func (e *ListEmailLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEmailLogsRequest, *apiresource.List[apiresource.EmailLog]] {
	return &apiendpoint.APIEndpoint[*ListEmailLogsRequest, *apiresource.List[apiresource.EmailLog]]{
		Title:             "List Email Logs",
		Description:       "Returns a paginated list of email logs for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/email-logs",
		Request:           &ListEmailLogsRequest{},
		Response:          &apiresource.List[apiresource.EmailLog]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeEmailLog,
			Fields:     []string{"sent_by"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEmailLogsRequest) (*apiresource.List[apiresource.EmailLog], *apierror.APIError) {
			return svc.(EmailLogSvc).ListEmailLogs
		},
	}
}
