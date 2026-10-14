package signingpinep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to set a temporary signing PIN for an account user.
type SetTemporarySigningPinRequest struct {
	// ID of the account user, in the caller's account, to set the PIN for.
	AccountUserID string `path:"id" validate:"required"`
	// The temporary PIN, following the same rules as any signing PIN.
	Pin string `json:"pin" validate:"required" sensitive:"true"`
}

var sampleSetTemporarySigningPinRequest = &SetTemporarySigningPinRequest{
	AccountUserID: apiresource.SampleAccountUserID,
	Pin:           samplePin,
}

func (*SetTemporarySigningPinRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetTemporarySigningPinRequest)
}

// Sets a temporary signing PIN for an account user who can't set their own, such as an operator who never signs in.
//
// The PIN can't be used to sign until the person replaces it with the change-signing-pin action. It replaces any PIN they had and clears any lockout.
type SetTemporarySigningPinEndpoint struct{}

func (e *SetTemporarySigningPinEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetTemporarySigningPinRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*SetTemporarySigningPinRequest, *apiresource.EmptyResource]{
		Title:               "Set Temporary Signing PIN",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users/{id}/actions/set-temporary-signing-pin",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetTemporarySigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SigningPinSvc).SetTemporarySigningPin
		},
	})
}
