package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to sync carrier options from Shippo.
type SyncOptionsRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
}

// Syncs carrier options from Shippo service levels, adding new and removing stale ones. Not available in sandbox mode.
type SyncOptionsEndpoint struct{}

func (e *SyncOptionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*SyncOptionsRequest, *apiresource.Carrier] {
	return (&apiendpoint.APIEndpoint[*SyncOptionsRequest, *apiresource.Carrier]{
		Title:             "Sync Carrier Options",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{id}/actions/sync-options",
		Request:           &SyncOptionsRequest{},
		Response:          &apiresource.Carrier{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SyncOptionsRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).SyncOptions
		},
	}).WithDocSource(e)
}
