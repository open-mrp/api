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
	Domain string `json:"domain" validate:"required"`
}

var sampleCreateEmailDomainRequest = &CreateEmailDomainRequest{
	Domain: "support.acme.com",
}

func (*CreateEmailDomainRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateEmailDomainRequest)
}

// Registers a customer-owned domain with the email bridge and returns the DKIM records to publish.
//
// The domain starts in `pending` until verified.
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
