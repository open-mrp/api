package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateAccountUserPasswordRequest is the request to update an account user's password.
type UpdateAccountUserPasswordRequest struct {
	// The ID of the account user whose password to update.
	AccountUserID string `path:"id" validate:"required"`
	// The requester's current password for verification.
	RequesterPassword string `json:"requester_password" validate:"required,password,max=255"`
	// The new password to set for the account user.
	NewPassword string `json:"new_password" validate:"required,password,max=255"`
}

var sampleUpdateAccountUserPasswordRequest = &UpdateAccountUserPasswordRequest{
	RequesterPassword: apiresource.SampleUserPassword,
	NewPassword:       apiresource.SampleNewUserPassword,
}

func (*UpdateAccountUserPasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountUserPasswordRequest)
}

type UpdateAccountUserPasswordEndpoint struct{}

func (e *UpdateAccountUserPasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountUserPasswordRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*UpdateAccountUserPasswordRequest, *apiresource.EmptyResource]{
		Title:             "Update Account User Password",
		Description:       "Updates an account user's password, requiring the requester's current password for verification.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}/password",
		Request:           &UpdateAccountUserPasswordRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			ShieldRequestBody: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountUserPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).UpdateAccountUserPassword
		},
	}
}
