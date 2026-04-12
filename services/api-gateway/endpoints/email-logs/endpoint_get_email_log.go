package emaillogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetEmailLogRequest is the request to retrieve a single email log.
type GetEmailLogRequest struct {
	// The ID of the email log to retrieve.
	EmailLogID string `path:"id" validate:"required"`
}

type GetEmailLogEndpoint struct{}

func (e *GetEmailLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetEmailLogRequest, *apiresource.EmailLog] {
	return &apiendpoint.APIEndpoint[*GetEmailLogRequest, *apiresource.EmailLog]{
		Title:             "Get Email Log",
		Description:       "Returns a single email log by its ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/email-logs/{id}",
		Request:           &GetEmailLogRequest{},
		Response:          &apiresource.EmailLog{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeEmailLog,
			Fields:     []string{"sent_by"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetEmailLogRequest) (*apiresource.EmailLog, *apierror.APIError) {
			return svc.(EmailLogSvc).GetEmailLog
		},
	}
}
