package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a DC location.
type DeleteDCLocationRequest struct {
	// DC location ID.
	DCLocationID string `path:"id" validate:"required"`
}

// Deletes a DC location.
//
// Deletion is permanent. Deleting the same location again reports that it has already been deleted rather than succeeding silently.
type DeleteDCLocationEndpoint struct{}

func (e *DeleteDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteDCLocationRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteDCLocationRequest, *apiresource.EmptyResource]{
		Title:             "Delete DC Location",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/dc-locations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainEdiRuns, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteDCLocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).DeleteDCLocation
		},
	})
}
