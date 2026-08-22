package inventorychangelogep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list inventory change logs.
type ListInventoryChangeLogsRequest struct {
	apiresource.PaginationRequest
	// Restricts results to changes affecting these items.
	ItemIDs []string `query:"item_ids"`
	// Restricts results to these action types (`scan`, `user_action`, `system_action`, `user_correction`).
	ActionTypeCodes []string `query:"action_type_codes"`
	// Restricts results to changes made by these users.
	//
	// Changes that were recorded without a responsible user are excluded whenever this filter is set.
	ChangedByUserIDs []string `query:"changed_by_user_ids"`
	// Restricts results to change logs created on or after this timestamp.
	StartDate *time.Time `query:"starts_at"`
	// Restricts results to change logs created on or before this timestamp.
	EndDate *time.Time `query:"ends_at"`
}

// Returns a paginated list of inventory change logs, newest first.
//
// Filters combine with AND, while the values within a single filter combine with OR. The `q` search term matches on item SKU, responsible user name, and scanning station name.
type ListInventoryChangeLogsEndpoint struct{}

func (e *ListInventoryChangeLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInventoryChangeLogsRequest, *apiresource.List[apiresource.InventoryChangeLog]] {
	return (&apiendpoint.APIEndpoint[*ListInventoryChangeLogsRequest, *apiresource.List[apiresource.InventoryChangeLog]]{
		Title:             "List Inventory Change Logs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventory-change-logs",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeInventoryChangeLog,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInventoryLogs, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInventoryChangeLogsRequest) (*apiresource.List[apiresource.InventoryChangeLog], *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).ListInventoryChangeLogs
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryChangeLog,
			Fields:     []string{"item", "quantity", "quantity.unit", "responsible_user", "responsible_scanning_station"},
		}),
	})
}
