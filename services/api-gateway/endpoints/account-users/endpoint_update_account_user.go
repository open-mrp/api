package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update an account user.
type UpdateAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
	// User display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// User email address.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// Unique username (3–255 chars; letters, numbers, underscores, hyphens).
	Username field.Optional[string] `json:"username,omitzero" validate:"omitempty,username"`
	// Role assigned to the user.
	RoleID field.Clearable[string] `json:"role_id,omitzero" validate:"omitempty"`
	// Department assigned to the user.
	DepartmentID field.Clearable[string] `json:"department_id,omitzero" validate:"omitempty"`
	// Notification preferences to update (external targets only).
	Preferences []NotificationPreferenceItem `json:"preferences,omitzero"`
}

var sampleUpdateAccountUserName = apiresource.SampleUserName
var sampleUpdateAccountUserRoleID = apiresource.SampleRoleID
var sampleUpdateAccountUserDepartmentID = apiresource.SampleDepartmentID
var sampleUpdateAccountUserRequest = &UpdateAccountUserRequest{
	Name:         field.Some(sampleUpdateAccountUserName),
	RoleID:       field.Set(sampleUpdateAccountUserRoleID),
	DepartmentID: field.Set(sampleUpdateAccountUserDepartmentID),
}

func (*UpdateAccountUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountUserRequest)
}

// Partially updates an account user.
type UpdateAccountUserEndpoint struct{}

func (e *UpdateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountUserRequest, *apiresource.AccountUser] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Update Account User",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).UpdateAccountUser
		},
		ObjectType: constants.ObjectTypeAccountUser,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"user", "role", "department"},
		}),
	})
}
