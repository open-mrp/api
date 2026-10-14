package batchreleaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to check an operator in at a scanning station.
type CheckInOperatorRequest struct {
	// ID of the scanning station.
	ScanningStationID string `path:"id" validate:"required"`
	// ID of the account user checking in.
	AccountUserID string `json:"account_user_id" validate:"required"`
	// The operator's signing PIN.
	Pin string `json:"pin" validate:"required" sensitive:"true"`
}

var sampleCheckInOperatorRequest = &CheckInOperatorRequest{
	ScanningStationID: apiresource.SampleScanningStationID,
	AccountUserID:     apiresource.SampleAccountUserID,
	Pin:               samplePin,
}

func (*CheckInOperatorRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCheckInOperatorRequest)
}

// Checks an operator in at a scanning station, identified by their signing PIN.
//
// Batches scanned at the station record the operator until they check out, someone else checks in, or 12 hours pass. Checking in ends whoever was checked in before.
type CheckInOperatorEndpoint struct{}

func (e *CheckInOperatorEndpoint) Materialize() *apiendpoint.APIEndpoint[*CheckInOperatorRequest, *apiresource.OperatorCheckIn] {
	return (&apiendpoint.APIEndpoint[*CheckInOperatorRequest, *apiresource.OperatorCheckIn]{
		Title:             "Check In Operator",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}/operator-check-ins",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOperatorCheckIn,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainBatches, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CheckInOperatorRequest) (*apiresource.OperatorCheckIn, *apierror.APIError) {
			return svc.(BatchReleaseSvc).CheckInOperator
		},
	})
}
