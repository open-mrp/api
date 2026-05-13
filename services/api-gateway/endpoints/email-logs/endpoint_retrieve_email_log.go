package emaillogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an email log.
type RetrieveEmailLogRequest struct {
	// Email log ID.
	EmailLogID string `path:"id" validate:"required"`
}

type RetrieveEmailLogEndpoint struct{}

func (e *RetrieveEmailLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveEmailLogRequest, *apiresource.EmailLog] {
	return &apiendpoint.APIEndpoint[*RetrieveEmailLogRequest, *apiresource.EmailLog]{
		Title:             "Retrieve Email Log",
		Description:       "Returns an email log by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/email-logs/{id}",
		Request:           &RetrieveEmailLogRequest{},
		Response:          &apiresource.EmailLog{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeEmailLog,
			Fields:     []string{"sent_by"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveEmailLogRequest) (*apiresource.EmailLog, *apierror.APIError) {
			return svc.(EmailLogSvc).GetEmailLog
		},
	}
}
