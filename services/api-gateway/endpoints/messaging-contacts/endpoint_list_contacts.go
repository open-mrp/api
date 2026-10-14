package contactep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list the caller's messageable contacts (the messaging directory).
//
// `q` matches contact names as a case-insensitive substring.
type ListContactsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Lists the people the caller can start a conversation with.
//
// For a member of the account, this is everyone active in that account, including themselves — messaging yourself is allowed. A customer signed in to the portal instead gets one shared "Customer Service" contact rather than the individual staff of the account they are dealing with; messages to it are routed by the account's support routes.
//
// Blocking is not applied to the directory: someone you have blocked, or who has blocked you, is still listed even though a direct message with them cannot be opened.
//
// The directory is returned as a single unpaginated page capped at 100 names, so narrow it with `q` in an account with many people.
type ListContactsEndpoint struct{}

func (e *ListContactsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListContactsRequest, *apiresource.List[apiresource.Actor]] {
	return (&apiendpoint.APIEndpoint[*ListContactsRequest, *apiresource.List[apiresource.Actor]]{
		Title:             "List Messaging Contacts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/messaging/contacts",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeActor,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListContactsRequest) (*apiresource.List[apiresource.Actor], *apierror.APIError) {
			return svc.(ContactSvc).ListContacts
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeActor,
			Fields:     []string{"role"},
		}),
	})
}
