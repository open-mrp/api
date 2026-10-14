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

// Request to retrieve who is checked in at a scanning station.
type RetrieveActiveOperatorCheckInRequest struct {
	// ID of the scanning station.
	ScanningStationID string `path:"id" validate:"required"`
}

// Returns the operator checked in at a scanning station now.
//
// Returns 404 when nobody is checked in.
type RetrieveActiveOperatorCheckInEndpoint struct{}

func (e *RetrieveActiveOperatorCheckInEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveActiveOperatorCheckInRequest, *apiresource.OperatorCheckIn] {
	return (&apiendpoint.APIEndpoint[*RetrieveActiveOperatorCheckInRequest, *apiresource.OperatorCheckIn]{
		Title:             "Retrieve Active Operator Check-In",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}/operator-check-in",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOperatorCheckIn,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainBatches, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveActiveOperatorCheckInRequest) (*apiresource.OperatorCheckIn, *apierror.APIError) {
			return svc.(BatchReleaseSvc).GetActiveOperatorCheckIn
		},
	})
}
