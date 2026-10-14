package batchreleaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to check an operator out of a scanning station.
type CheckOutOperatorRequest struct {
	// ID of the scanning station.
	ScanningStationID string `path:"id" validate:"required"`
	// ID of the check-in to end.
	CheckInID string `path:"check_in_id" validate:"required"`
}

// Checks an operator out of a scanning station.
//
// Checking out a check-in that already ended returns it unchanged.
type CheckOutOperatorEndpoint struct{}

func (e *CheckOutOperatorEndpoint) Materialize() *apiendpoint.APIEndpoint[*CheckOutOperatorRequest, *apiresource.OperatorCheckIn] {
	return (&apiendpoint.APIEndpoint[*CheckOutOperatorRequest, *apiresource.OperatorCheckIn]{
		Title:             "Check Out Operator",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}/operator-check-ins/{check_in_id}/actions/check-out",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOperatorCheckIn,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainBatches, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CheckOutOperatorRequest) (*apiresource.OperatorCheckIn, *apierror.APIError) {
			return svc.(BatchReleaseSvc).CheckOutOperator
		},
	})
}
