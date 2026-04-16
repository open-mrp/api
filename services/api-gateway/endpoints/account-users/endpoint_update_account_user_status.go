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

// Request to transition an account user to a target status.
type UpdateAccountUserStatusRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
	// Target status.
	Status constants.AccountUserStatus `json:"status" validate:"required,oneof=active disabled removed"`
}

var sampleUpdateAccountUserStatusRequest = &UpdateAccountUserStatusRequest{
	Status: constants.AccountUserStatusDisabled,
}

func (*UpdateAccountUserStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountUserStatusRequest)
}

type UpdateAccountUserStatusEndpoint struct{}

func (e *UpdateAccountUserStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountUserStatusRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*UpdateAccountUserStatusRequest, *apiresource.EmptyResource]{
		Title:             "Update Account User Status",
		Description:       "Transitions an account user to a target status.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}/status",
		Request:           &UpdateAccountUserStatusRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountUserStatusRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).UpdateAccountUserStatus
		},
	}
}
