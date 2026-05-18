package auditeventsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the valid audit event resource types.
type ListAuditEventResourceTypesRequest struct{}

// Returns the full set of resource types that may appear on audit events.
type ListAuditEventResourceTypesEndpoint struct{}

func (e *ListAuditEventResourceTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAuditEventResourceTypesRequest, *apiresource.List[constants.ObjectType]] {
	return (&apiendpoint.APIEndpoint[*ListAuditEventResourceTypesRequest, *apiresource.List[constants.ObjectType]]{
		Title:             "List Audit Event Resource Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/audit-events/resource-types",
		Request:           &ListAuditEventResourceTypesRequest{},
		Response:          &apiresource.List[constants.ObjectType]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAuditEventResourceTypesRequest) (*apiresource.List[constants.ObjectType], *apierror.APIError) {
			return svc.(AuditEventSvc).ListAuditEventResourceTypes
		},
	}).WithDocSource(e)
}
