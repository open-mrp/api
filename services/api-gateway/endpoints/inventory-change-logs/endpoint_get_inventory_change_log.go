package inventorychangelogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetInventoryChangeLogRequest is the request to retrieve an inventory change log by ID.
type GetInventoryChangeLogRequest struct {
	// Inventory change log ID.
	InventoryChangeLogID string `path:"id" validate:"required"`
}

type GetInventoryChangeLogEndpoint struct{}

func (e *GetInventoryChangeLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetInventoryChangeLogRequest, *apiresource.InventoryChangeLog] {
	return &apiendpoint.APIEndpoint[*GetInventoryChangeLogRequest, *apiresource.InventoryChangeLog]{
		Title:             "Get Inventory Change Log",
		Description:       "Returns an inventory change log by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventory-change-logs/{id}",
		Request:           &GetInventoryChangeLogRequest{},
		Response:          &apiresource.InventoryChangeLog{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetInventoryChangeLogRequest) (*apiresource.InventoryChangeLog, *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).GetInventoryChangeLog
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryChangeLog,
			Fields:     []string{"item", "quantity", "quantity.unit", "responsible_user", "responsible_scanning_station"},
		}),
	}
}
