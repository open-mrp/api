package emaillogep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list email logs.
type ListEmailLogsRequest struct {
	object.ListRequest
	apirequest.CreatedRange
	// Case-insensitive substring match on `subject` or any recipient address.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return email logs with any of these send statuses: `sent` or `pending`.
	SendStatuses []constants.EmailSendStatus `query:"send_statuses"`
	// Only return email logs that carried any of these records, such as an invoice ID.
	RelatedIDs []string `query:"related_ids"`
}

// Returns a list of email logs.
//
// Sorted by creation date, newest first.
type ListEmailLogsEndpoint struct{}

func (e *ListEmailLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEmailLogsRequest, *apiresource.List[apiresource.EmailLog]] {
	return (&apiendpoint.APIEndpoint[*ListEmailLogsRequest, *apiresource.List[apiresource.EmailLog]]{
		Title:             "List Email Logs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/email-logs",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainEmailLogs, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEmailLogsRequest) (*apiresource.List[apiresource.EmailLog], *apierror.APIError) {
			return svc.(EmailLogSvc).ListEmailLogs
		},
		ObjectType: constants.ObjectTypeEmailLog,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeEmailLog,
			Fields:     []string{"related.record"},
		}),
	})
}
