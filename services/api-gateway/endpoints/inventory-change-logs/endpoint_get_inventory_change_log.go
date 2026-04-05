package inventorychangelogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetInventoryChangeLogRequest is the request to retrieve a single inventory change log by ID.
type GetInventoryChangeLogRequest struct {
	// The ID of the inventory change log to retrieve.
	InventoryChangeLogID string `path:"id" validate:"required"`
}

type GetInventoryChangeLogEndpoint struct{}

func (e *GetInventoryChangeLogEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetInventoryChangeLogRequest, *apiresource.InventoryChangeLog] {
	return &apiendpoint.APIEndpoint[*GetInventoryChangeLogRequest, *apiresource.InventoryChangeLog]{
		Title:             "Get Inventory Change Log",
		Description:       "Returns a single inventory change log by its ID.",
		Method:            http.MethodGet,
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
