package contactep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the caller's messageable contacts (the messaging directory).
type ListContactsRequest struct {
	apiresource.PaginationRequest
}

// Lists the caller's messageable contacts.
type ListContactsEndpoint struct{}

func (e *ListContactsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListContactsRequest, *apiresource.List[apiresource.Actor]] {
	return (&apiendpoint.APIEndpoint[*ListContactsRequest, *apiresource.List[apiresource.Actor]]{
		Title:               "List Messaging Contacts",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/contacts",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeActor,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListContactsRequest) (*apiresource.List[apiresource.Actor], *apierror.APIError) {
			return svc.(ContactSvc).ListContacts
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeActor,
			Fields:     []string{"role"},
		}),
	})
}
