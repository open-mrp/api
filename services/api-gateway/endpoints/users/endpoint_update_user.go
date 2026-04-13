package userep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a user.
type UpdateUserRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name" nullable:"false" validate:"omitempty,max=255"`
	// Profile image URL.
	ImageUrl *string `json:"image_url" nullable:"false" validate:"omitempty,max=2083"`
	// Email verification timestamp. Set to null to mark as unverified.
	EmailVerified *time.Time `json:"email_verified" nullable:"false"`
}

var sampleUpdateUserName = apiresource.SampleUserName
var sampleUpdateUserImageUrl = apiresource.SampleUserImageUrl
var sampleUpdateUserRequest = &UpdateUserRequest{
	Name:     &sampleUpdateUserName,
	ImageUrl: &sampleUpdateUserImageUrl,
}

func (*UpdateUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUserRequest)
}

type UpdateUserEndpoint struct{}

func (e *UpdateUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUserRequest, *apiresource.User] {
	return &apiendpoint.APIEndpoint[*UpdateUserRequest, *apiresource.User]{
		Title:             "Update User",
		Description:       "Partially updates a user's profile.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/identity/users/{id}",
		Request:           &UpdateUserRequest{},
		Response:          &apiresource.User{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUserRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(UserSvc).UpdateUser
		},
	}
}
