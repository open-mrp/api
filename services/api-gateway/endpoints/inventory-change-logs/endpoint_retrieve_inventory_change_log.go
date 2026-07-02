package inventorychangelogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveInventoryChangeLogRequest is the request to retrieve an inventory change log by ID.
type RetrieveInventoryChangeLogRequest struct {
	// Inventory change log ID.
	InventoryChangeLogID string `path:"id" validate:"required"`
}

// Returns an inventory change log by ID.
type RetrieveInventoryChangeLogEndpoint struct{}

func (e *RetrieveInventoryChangeLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveInventoryChangeLogRequest, *apiresource.InventoryChangeLog] {
	return (&apiendpoint.APIEndpoint[*RetrieveInventoryChangeLogRequest, *apiresource.InventoryChangeLog]{
		Title:             "Retrieve Inventory Change Log",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventory-change-logs/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeInventoryChangeLog,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInventoryLogs, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveInventoryChangeLogRequest) (*apiresource.InventoryChangeLog, *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).GetInventoryChangeLog
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryChangeLog,
			Fields:     []string{"item", "quantity", "quantity.unit", "responsible_user", "responsible_scanning_station"},
		}),
	})
}
