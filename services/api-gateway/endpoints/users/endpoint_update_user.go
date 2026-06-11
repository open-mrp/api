package userep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a user.
type UpdateUserRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Profile image URL.
	ImageUrl field.Optional[string] `json:"image_url,omitzero" validate:"omitempty,max=2083"`
	// Timestamp recording when the user's email address was verified.
	EmailVerified field.Optional[time.Time] `json:"email_verified,omitzero"`
}

var sampleUpdateUserName = apiresource.SampleUserName
var sampleUpdateUserImageUrl = apiresource.SampleUserImageUrl
var sampleUpdateUserRequest = &UpdateUserRequest{
	Name:     field.Some(sampleUpdateUserName),
	ImageUrl: field.Some(sampleUpdateUserImageUrl),
}

func (*UpdateUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUserRequest)
}

// Partially updates a user's profile.
type UpdateUserEndpoint struct{}

func (e *UpdateUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUserRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*UpdateUserRequest, *apiresource.User]{
		Title:             "Update User",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/identity/users/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUser,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUserRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(UserSvc).UpdateUser
		},
	})
}
