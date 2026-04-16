package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a scanner-role account user's password.
type UpdateScannerPasswordRequest struct {
	// Target scanner account user ID.
	AccountUserID string `json:"account_user_id" validate:"required,max=191"`
	// Requester's current password (the caller's own password, for verification).
	RequesterPassword string `json:"requester_password" validate:"required,password,max=255"`
	// New password to set for the scanner user.
	NewPassword string `json:"new_password" validate:"required,password,max=255"`
}

var sampleUpdateScannerPasswordRequest = &UpdateScannerPasswordRequest{
	AccountUserID:     apiresource.SampleAccountUserID,
	RequesterPassword: apiresource.SampleUserPassword,
	NewPassword:       apiresource.SampleNewUserPassword,
}

func (*UpdateScannerPasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateScannerPasswordRequest)
}

type UpdateScannerPasswordEndpoint struct{}

func (e *UpdateScannerPasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateScannerPasswordRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*UpdateScannerPasswordRequest, *apiresource.EmptyResource]{
		Title:             "Update Scanner Password",
		Description:       "Rotates the password for a scanner-role account user backing a scanning station. Requires the caller's current password for verification.",
		Method:            http.MethodPost,
		Route:             "/v1/auth/scanner-passwords",
		ContentType:       "application/json",
		Request:           &UpdateScannerPasswordRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			ShieldRequestBody: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateScannerPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).UpdateScannerPassword
		},
	}
}
