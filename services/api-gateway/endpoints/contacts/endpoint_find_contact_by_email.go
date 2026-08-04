package contactep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to find contacts by email.
type FindContactByEmailRequest struct {
	// The email address to look up.
	Email string `json:"email" validate:"required,email"`
	// Restricts the results to matches whose relationship to your account is one of these.
	//
	// Leaving it out returns matches of every relationship.
	Relationships []constants.ContactRelationship `query:"relationships"`
}

var sampleFindContactByEmailRequest = &FindContactByEmailRequest{
	Email: "buyer@acme-co.example",
}

func (*FindContactByEmailRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleFindContactByEmailRequest)
}

// Finds the contacts that match an email address.
//
// Only active people on accounts you have a relationship with are returned — your customers, your suppliers, or your own account. A match's `relationship` says how you relate to the account it belongs to. The same person can be set up on several accounts under one email, so this can return more than one match, and an email that belongs to no one you deal with simply returns no matches rather than an error.
type FindContactByEmailEndpoint struct{}

func (e *FindContactByEmailEndpoint) Materialize() *apiendpoint.APIEndpoint[*FindContactByEmailRequest, *apiresource.List[apiresource.ContactMatch]] {
	return (&apiendpoint.APIEndpoint[*FindContactByEmailRequest, *apiresource.List[apiresource.ContactMatch]]{
		Title:             "Find Contact by Email",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/contacts/actions/find-by-email",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeContactMatch,
		ServiceHandler: func(svc any) func(ctx context.Context, req *FindContactByEmailRequest) (*apiresource.List[apiresource.ContactMatch], *apierror.APIError) {
			return svc.(ContactSvc).FindContactByEmail
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeContactMatch,
			Fields: []string{
				"account_user",
				"account_user.user",
				"account_user.role",
				"account_user.department",
				"account",
			},
		}),
	})
}
