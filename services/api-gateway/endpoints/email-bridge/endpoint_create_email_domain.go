package emailbridgeep

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

// Request to register a sending/receiving domain with the email bridge.
type CreateEmailDomainRequest struct {
	// The fully-qualified domain name to register (e.g. `support.acme.com`).
	//
	// Supply a bare domain, not an email address; the value is lowercased before it is stored.
	Domain string `json:"domain" validate:"required"`
}

var sampleCreateEmailDomainRequest = &CreateEmailDomainRequest{
	Domain: "support.acme.com",
}

func (*CreateEmailDomainRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateEmailDomainRequest)
}

// Registers a domain you own with the email bridge and returns the DKIM tokens to publish.
//
// The domain starts in `pending`. Publish each returned token as a CNAME record in the domain's DNS, then call the verify action to move it to `verified`; only then can inboxes be created on it.
//
// A domain can only be registered once across the platform, so registering one that is already in use returns a conflict error.
type CreateEmailDomainEndpoint struct{}

func (e *CreateEmailDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateEmailDomainRequest, *apiresource.EmailDomain] {
	return (&apiendpoint.APIEndpoint[*CreateEmailDomainRequest, *apiresource.EmailDomain]{
		Title:               "Create Email Domain",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-domains",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError) {
			return svc.(EmailBridgeSvc).CreateDomain
		},
	})
}
