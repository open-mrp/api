package emaillogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an email log.
type RetrieveEmailLogRequest struct {
	// Email log ID.
	EmailLogID string `path:"id" validate:"required"`
}

// Returns an email log by ID.
type RetrieveEmailLogEndpoint struct{}

func (e *RetrieveEmailLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveEmailLogRequest, *apiresource.EmailLog] {
	return (&apiendpoint.APIEndpoint[*RetrieveEmailLogRequest, *apiresource.EmailLog]{
		Title:               "Retrieve Email Log",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/core/email-logs/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainEmailLogs, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveEmailLogRequest) (*apiresource.EmailLog, *apierror.APIError) {
			return svc.(EmailLogSvc).GetEmailLog
		},
		ObjectType: constants.ObjectTypeEmailLog,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeEmailLog,
			Fields:     []string{"sent_by"},
		}),
	})
}
