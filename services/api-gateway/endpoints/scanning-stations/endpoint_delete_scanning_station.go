package scanningstationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a scanning station.
type DeleteScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
}

// Deletes a scanning station.
//
// A station that batches, production steps or inventory changes name can't be deleted (409 `resource_in_use`).
type DeleteScanningStationEndpoint struct{}

func (e *DeleteScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteScanningStationRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteScanningStationRequest, *object.Deleted]{
		Title:             "Delete Scanning Station",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeScanningStation,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainScanningStations, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteScanningStationRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ScanningStationSvc).DeleteScanningStation, constants.ObjectTypeScanningStation, "id")
		},
	})
}
