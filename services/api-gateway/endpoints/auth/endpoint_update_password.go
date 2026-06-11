package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a user's password.
type UpdatePasswordRequest struct {
	// Current password.
	OldPassword string `json:"old_password" validate:"required,password,max=255" sensitive:"true"`
	// New password.
	NewPassword string `json:"new_password" validate:"required,password,max=255" sensitive:"true"`
}

var sampleUpdatePasswordRequest = &UpdatePasswordRequest{
	OldPassword: apiresource.SampleUserPassword,
	NewPassword: apiresource.SampleNewUserPassword,
}

func (*UpdatePasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePasswordRequest)
}

// Updates the authenticated user's password after verifying their current password.
//
// All of the user's existing refresh tokens are revoked, signing out their other active sessions.
type UpdatePasswordEndpoint struct{}

func (e *UpdatePasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePasswordRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UpdatePasswordRequest, *apiresource.EmptyResource]{
		Title:             "Update Password",
		Method:            http.MethodPost,
		Route:             "/v1/auth/passwords",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).UpdatePassword
		},
	})
}
