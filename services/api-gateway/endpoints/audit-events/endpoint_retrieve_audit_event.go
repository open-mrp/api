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
type RetrieveAuditEventRequest struct {
	// Audit event ID.
	ID string `path:"id" validate:"required"`
}

// Returns an audit event by ID.
type RetrieveAuditEventEndpoint struct{}

func (e *RetrieveAuditEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAuditEventRequest, *apiresource.AuditEvent] {
	return (&apiendpoint.APIEndpoint[*RetrieveAuditEventRequest, *apiresource.AuditEvent]{
		Title:             "Retrieve Audit Event",
		Method:            http.MethodGet,
		Route:             "/v1/core/audit-events/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAuditEvent,
			Fields:     []string{"actor", "changes", "metadata", "request"},
		}),
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAuditEventRequest) (*apiresource.AuditEvent, *apierror.APIError) {
			return svc.(AuditEventSvc).GetAuditEvent
		},
	})
}
