package auditeventsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an audit event.
type GetAuditEventRequest struct {
	// Audit event ID.
	ID string `path:"id" validate:"required"`
}

type GetAuditEventEndpoint struct{}

func (e *GetAuditEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAuditEventRequest, *apiresource.AuditEvent] {
	return &apiendpoint.APIEndpoint[*GetAuditEventRequest, *apiresource.AuditEvent]{
		Title:             "Get Audit Event",
		Description:       "Returns an audit event by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/core/audit-events/{id}",
		ContentType:       "application/json",
		Request:           &GetAuditEventRequest{},
		Response:          &apiresource.AuditEvent{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAuditEvent,
			Fields:     []string{"actor", "changes", "metadata"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAuditEventRequest) (*apiresource.AuditEvent, *apierror.APIError) {
			return svc.(AuditEventSvc).GetAuditEvent
		},
	}
}
