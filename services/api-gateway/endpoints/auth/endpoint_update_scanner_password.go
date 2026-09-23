package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to set the password of an account user who has no email address.
type UpdateScannerPasswordRequest struct {
	// ID of the account user whose password is being changed.
	//
	// Must belong to the caller's account, have no email address, and not hold the admin role; requests targeting any other user are rejected.
	AccountUserID string `json:"account_user_id" validate:"required"`
	// The caller's own current password, used to confirm the caller's identity before the user's password is changed.
	RequesterPassword string `json:"requester_password" validate:"required,password,max=255" sensitive:"true"`
	// New password to set for the user.
	NewPassword string `json:"new_password" validate:"required,password,max=255" sensitive:"true"`
}

var sampleUpdateScannerPasswordRequest = &UpdateScannerPasswordRequest{
	AccountUserID:     apiresource.SampleAccountUserID,
	RequesterPassword: apiresource.SampleUserPassword,
	NewPassword:       apiresource.SampleNewUserPassword,
}

func (*UpdateScannerPasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateScannerPasswordRequest)
}

// Sets a new password for an account user who signs in with a username and has no email address, such as a scanning station login.
//
// The caller must be signed in as a user with permission to manage team users and must supply their own current password; API keys cannot perform this operation because they have no password to verify. Only non-admin users in the caller's account without an email address can be changed this way, since they cannot receive a password reset email — users with an email address must use the password reset flow.
type UpdateScannerPasswordEndpoint struct{}

func (e *UpdateScannerPasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateScannerPasswordRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UpdateScannerPasswordRequest, *apiresource.EmptyResource]{
		Title:             "Set Username User Password",
		Method:            http.MethodPost,
		Route:             "/v1/auth/scanner-passwords",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateScannerPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).UpdateScannerPassword
		},
	})
}
