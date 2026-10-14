package signingpinep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to change an account user's signing PIN.
type ChangeSigningPinRequest struct {
	// ID of the account user, in the caller's account, whose PIN to change.
	AccountUserID string `path:"id" validate:"required"`
	// The person's current PIN, temporary or not.
	CurrentPin string `json:"current_pin" validate:"required" sensitive:"true"`
	// The new PIN: 4 to 8 digits, different from the current one, not one repeated digit and not a run of consecutive digits.
	NewPin string `json:"new_pin" validate:"required" sensitive:"true"`
}

var sampleChangeSigningPinRequest = &ChangeSigningPinRequest{
	AccountUserID: apiresource.SampleAccountUserID,
	CurrentPin:    samplePin,
	NewPin:        sampleNewPin,
}

func (*ChangeSigningPinRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleChangeSigningPinRequest)
}

// Replaces an account user's signing PIN after checking their current one. This is how a temporary PIN is replaced, and it can be called from a shared station.
//
// A wrong current PIN counts toward the lockout like a wrong PIN at signing: five in a row lock signing for that person for 15 minutes.
type ChangeSigningPinEndpoint struct{}

func (e *ChangeSigningPinEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeSigningPinRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ChangeSigningPinRequest, *apiresource.EmptyResource]{
		Title:             "Change Signing PIN",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}/actions/change-signing-pin",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SigningPinSvc).ChangeSigningPin
		},
	})
}
