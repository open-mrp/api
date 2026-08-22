package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a DC location.
type RetrieveDCLocationRequest struct {
	// DC location ID.
	DCLocationID string `path:"id" validate:"required"`
}

// Returns a DC location by ID.
type RetrieveDCLocationEndpoint struct{}

func (e *RetrieveDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveDCLocationRequest, *apiresource.DCLocation] {
	return (&apiendpoint.APIEndpoint[*RetrieveDCLocationRequest, *apiresource.DCLocation]{
		Title:             "Retrieve DC Location",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/dc-locations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainEdiRuns, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeDCLocation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).GetDCLocation
		},
	})
}
