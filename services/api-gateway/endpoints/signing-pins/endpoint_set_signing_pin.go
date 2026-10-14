package signingpinep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to set the caller's own signing PIN.
type SetSigningPinRequest struct {
	// The new PIN: 4 to 8 digits, not one repeated digit and not a run of consecutive digits such as 1234 or 9876.
	Pin string `json:"pin" validate:"required" sensitive:"true"`
}

var sampleSetSigningPinRequest = &SetSigningPinRequest{Pin: samplePin}

func (*SetSigningPinRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetSigningPinRequest)
}

// Sets the signed-in user's signing PIN, used to sign batch releases and to check in at scanning stations.
//
// Replaces any PIN the user had, including a temporary one, and clears any lockout from wrong PINs. The caller must be signed in as themselves: API keys, agents and scanning station logins can't set a PIN. Nobody can read a PIN back.
type SetSigningPinEndpoint struct{}

func (e *SetSigningPinEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetSigningPinRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*SetSigningPinRequest, *apiresource.EmptyResource]{
		Title:             "Set Signing PIN",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/users/me/signing-pin",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SigningPinSvc).SetSigningPin
		},
	})
}
