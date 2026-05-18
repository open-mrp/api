package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update an account user.
type UpdateAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
	// User display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// User email address.
	Email *string `json:"email,omitempty" nullable:"false" validate:"omitnil,custom_email,max=255"`
	// Unique username (3–255 chars; letters, numbers, underscores, hyphens).
	Username *string `json:"username,omitempty" nullable:"false" validate:"omitempty,username"`
	// Role assigned to the user.
	RoleID *string `json:"role_id,omitempty" nullable:"true" validate:"omitempty"`
	// Department assigned to the user.
	DepartmentID *string `json:"department_id,omitempty" nullable:"true" validate:"omitempty"`
	// Notification preferences to update (external targets only).
	Preferences []NotificationPreferenceItem `json:"preferences,omitempty"`
}

var sampleUpdateAccountUserName = apiresource.SampleUserName
var sampleUpdateAccountUserRoleID = apiresource.SampleRoleID
var sampleUpdateAccountUserDepartmentID = apiresource.SampleDepartmentID
var sampleUpdateAccountUserRequest = &UpdateAccountUserRequest{
	Name:         &sampleUpdateAccountUserName,
	RoleID:       &sampleUpdateAccountUserRoleID,
	DepartmentID: &sampleUpdateAccountUserDepartmentID,
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
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"role", "department"},
		}),
	})
}
