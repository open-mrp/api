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
	OldPassword string `json:"old_password" validate:"required,password,max=255"`
	// New password.
	NewPassword string `json:"new_password" validate:"required,password,max=255"`
}

var sampleUpdatePasswordRequest = &UpdatePasswordRequest{
	OldPassword: apiresource.SampleUserPassword,
	NewPassword: apiresource.SampleNewUserPassword,
}

func (*UpdatePasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePasswordRequest)
}

// Updates a user's password, revoking previous tokens and setting new access and refresh tokens in cookies.
type UpdatePasswordEndpoint struct{}

func (e *UpdatePasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePasswordRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UpdatePasswordRequest, *apiresource.EmptyResource]{
		Title:             "Update Password",
		Method:            http.MethodPost,
		Route:             "/v1/auth/passwords",
		ContentType:       "application/json",
		Request:           &UpdatePasswordRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).UpdatePassword
		},
		Extras: apiendpoint.APIEndpointExtras{
			ShieldRequestBody: true,
		},
	}).WithDocSource(e)
}
