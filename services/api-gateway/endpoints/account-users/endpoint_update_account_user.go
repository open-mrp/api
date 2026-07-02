package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
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
	//
	// Must not already be in use by another user.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// Unique username.
	//
	// 3–255 characters; letters, numbers, underscores, and hyphens. Must not already be in use by another user.
	Username field.Optional[string] `json:"username,omitzero" validate:"omitempty,username"`
	// ID of the role to assign to the user.
	//
	// Set to `null` to clear the role.
	RoleID field.Clearable[string] `json:"role_id,omitzero" validate:"omitempty"`
	// ID of the department to assign to the user.
	//
	// Set to `null` to clear the department.
	DepartmentID field.Clearable[string] `json:"department_id,omitzero" validate:"omitempty"`
	// Notification preference toggles to apply.
	//
	// Only allowed when updating a user in another account you manage (cross-account); rejected otherwise. Notification types omitted from the list are left unchanged.
	Preferences []NotificationPreferenceItem `json:"preferences,omitzero"`
}

var sampleUpdateAccountUserName = apiresource.SampleUserName
var sampleUpdateAccountUserEmail = apiresource.SampleUserEmail
var sampleUpdateAccountUserUsername = apiresource.SampleUserUsername
var sampleUpdateAccountUserRoleID = apiresource.SampleRoleID
var sampleUpdateAccountUserDepartmentID = apiresource.SampleDepartmentID
var sampleUpdateAccountUserRequest = &UpdateAccountUserRequest{
	Name:         field.Some(sampleUpdateAccountUserName),
	Email:        field.Some(sampleUpdateAccountUserEmail),
	Username:     field.Some(sampleUpdateAccountUserUsername),
	RoleID:       field.Set(sampleUpdateAccountUserRoleID),
	DepartmentID: field.Set(sampleUpdateAccountUserDepartmentID),
	Preferences: []NotificationPreferenceItem{
		{NotificationTypeCode: constants.AccountRelationNotificationTypeOrderAcknowledgement, Enabled: true},
	},
}

func (*UpdateAccountUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountUserRequest)
}

// Partially updates an account user.
//
// Omitted fields are left unchanged. Profile fields (`name`, `email`, `username`) update the underlying user, which is shared across every account the user belongs to.
type UpdateAccountUserEndpoint struct{}

func (e *UpdateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountUserRequest, *apiresource.AccountUser] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountUserRequest, *apiresource.AccountUser]{
		Title:               "Update Account User",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
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
