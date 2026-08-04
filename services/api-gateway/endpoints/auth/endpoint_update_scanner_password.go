package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a scanner-role account user's password.
type UpdateScannerPasswordRequest struct {
	// ID of the account user whose password is being changed.
	//
	// Must belong to the caller's account and hold a scanner role; requests targeting any other user are rejected.
	AccountUserID string `json:"account_user_id" validate:"required"`
	// The caller's own current password, used to confirm the caller's identity before the scanner password is changed.
	RequesterPassword string `json:"requester_password" validate:"required,password,max=255" sensitive:"true"`
	// New password to set for the scanner user.
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

// Sets a new password for a scanner-role account user, the login used by a scanning station.
//
// The caller must be signed in as a user with permission to manage team users and must supply their own current password; API keys cannot perform this operation because they have no password to verify. Only scanner-role users in the caller's account can be changed this way — use the password reset flow for everyone else.
type UpdateScannerPasswordEndpoint struct{}

func (e *UpdateScannerPasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateScannerPasswordRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UpdateScannerPasswordRequest, *apiresource.EmptyResource]{
		Title:             "Update Scanner Password",
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
