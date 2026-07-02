package inventorychangelogep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListInventoryChangeLogsRequest is the request to list inventory change logs.
type ListInventoryChangeLogsRequest struct {
	apiresource.PaginationRequest
	// Filter by item IDs.
	ItemIDs []string `query:"item_ids"`
	// Filter by the action that produced the change.
	ActionTypeCodes []string `query:"action_type_codes"`
	// Filter by the user responsible for the change.
	ChangedByUserIDs []string `query:"changed_by_user_ids"`
	// Restricts results to change logs created on or after this timestamp.
	StartDate *time.Time `query:"start_date"`
	// Restricts results to change logs created on or before this timestamp.
	EndDate *time.Time `query:"end_date"`
}

// Returns a paginated list of inventory change logs, newest first.
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
