package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to sync a carrier's service levels from Shippo.
type SyncOptionsRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
}

// Re-syncs a carrier's service levels from Shippo.
//
// Service levels newly offered by the carrier are added (initially hidden from the customer portal) and previously synced ones no longer offered are removed; manually created service levels are untouched. Only available for Shippo-managed carriers; not available in sandbox mode.
type SyncOptionsEndpoint struct{}

func (e *SyncOptionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*SyncOptionsRequest, *apiresource.Carrier] {
	return (&apiendpoint.APIEndpoint[*SyncOptionsRequest, *apiresource.Carrier]{
		Title:               "Sync Carrier Service Levels",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/carriers/{id}/actions/sync-options",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeCarrier,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SyncOptionsRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).SyncOptions
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"owner", "owner.account", "service_levels"},
		}),
	})
}
