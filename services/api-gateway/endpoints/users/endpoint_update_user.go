package userep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a user.
type UpdateUserRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
	// The user's full display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Location of the user's profile image.
	//
	// Uploading a photo through Upload User Photo overwrites whatever is set here.
	ImageUrl field.Optional[string] `json:"image_url,omitzero" validate:"omitempty,max=2083"`
	// When the user's email address was verified.
	//
	// Setting this marks the address as verified outright; no verification email is sent and no verification link is checked.
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

// Updates a user's global profile.
//
// Changes apply everywhere the user appears, in every account they belong to. Account-specific details such as their status, role, and department are changed on the account user record instead.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUserRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(UserSvc).UpdateUser
		},
	})
}
