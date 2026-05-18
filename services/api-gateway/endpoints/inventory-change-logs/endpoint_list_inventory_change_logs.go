package inventorychangelogep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListInventoryChangeLogsRequest is the request to list inventory change logs.
type ListInventoryChangeLogsRequest struct {
	apiresource.PaginationRequest
	// Filter by item IDs.
	ItemIDs []string `query:"item_ids"`
	// Filter by action type codes.
	ActionTypeCodes []string `query:"action_type_codes"`
	// Filter by responsible user IDs.
	ChangedByUserIDs []string `query:"changed_by_user_ids"`
	// Filter change logs created on or after this date.
	StartDate *time.Time `query:"start_date"`
	// Filter change logs created on or before this date.
	EndDate *time.Time `query:"end_date"`
}

// Returns a paginated list of inventory change logs for the target account.
type ListInventoryChangeLogsEndpoint struct{}

func (e *ListInventoryChangeLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInventoryChangeLogsRequest, *apiresource.List[apiresource.InventoryChangeLog]] {
	return (&apiendpoint.APIEndpoint[*ListInventoryChangeLogsRequest, *apiresource.List[apiresource.InventoryChangeLog]]{
		Title:             "List Inventory Change Logs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventory-change-logs",
		Request:           &ListInventoryChangeLogsRequest{},
		Response:          &apiresource.List[apiresource.InventoryChangeLog]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInventoryChangeLogsRequest) (*apiresource.List[apiresource.InventoryChangeLog], *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).ListInventoryChangeLogs
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryChangeLog,
			Fields:     []string{"item", "quantity", "quantity.unit", "responsible_user", "responsible_scanning_station"},
		}),
	}).WithDocSource(e)
}
