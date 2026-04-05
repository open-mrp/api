package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateAccountUserRequest is the request to partially update an account user.
type UpdateAccountUserRequest struct {
	// The ID of the account user to update.
	AccountUserID string `path:"id" validate:"required"`
	// The user's display name.
	Name *string `json:"name,omitempty"`
	// The user's email address.
	Email *string `json:"email,omitempty" validate:"omitnil,custom_email"`
	// The user's username.
	Username *string `json:"username,omitempty"`
	// The ID of the role to assign.
	RoleID *string `json:"role_id,omitempty" nullable:"true"`
	// The ID of the department to assign.
	DepartmentID *string `json:"department_id,omitempty" nullable:"true"`
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

type UpdateAccountUserEndpoint struct{}

func (e *UpdateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountUserRequest, *apiresource.AccountUser] {
	return &apiendpoint.APIEndpoint[*UpdateAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Update Account User",
		Description:       "Partially updates an account user.",
		Method:            http.MethodPatch,
		Route:             "/v1/identity/account-users/{id}",
		Request:           &UpdateAccountUserRequest{},
		Response:          &apiresource.AccountUser{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).UpdateAccountUser
		},
	}
}
