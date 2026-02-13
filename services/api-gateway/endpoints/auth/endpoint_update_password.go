package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to update a user's password
type UpdatePasswordRequest struct {
	// The user's current password
	OldPassword string `json:"old_password" validate:"required,password"`
	// The new password to be set
	NewPassword string `json:"new_password" validate:"required,password"`
}

var sampleUpdatePasswordRequest = &UpdatePasswordRequest{
	OldPassword: apiresource.SampleUserPassword,
	NewPassword: apiresource.SampleNewUserPassword,
}

func (*UpdatePasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePasswordRequest)
}

const updatePasswordEndpointDescription string = `This endpoint is used to create a new password for a user. Once completed, new access and refresh tokens 
are set in cookies, and previous tokens are revoked.`

type UpdatePasswordEndpoint struct{}

func (e *UpdatePasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePasswordRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*UpdatePasswordRequest, *apiresource.EmptyResource]{
		Title:             "Create New Password",
		Description:       updatePasswordEndpointDescription,
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
			AllowUnknownJSONFields: false,
		},
	}
}
