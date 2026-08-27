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
	// Opaque cursor token identifying where the page of results starts.
	//
	// Use the `cursor` value embedded in a previous response's `next_page_url` or `previous_page_url` to fetch the adjacent page. Omit to start from the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of results to return in a single page.
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
	// Restricts results to changes affecting these items.
	ItemIDs []string `query:"item_ids"`
	// Restricts results to these action types.
	ActionTypes []constants.InventoryActionType `query:"action_types"`
	// Restricts results to changes made by these users.
	//
	// Changes that were recorded without a responsible user are excluded whenever this filter is set.
	ChangedByUserIDs []string `query:"changed_by_user_ids"`
	// Restricts results to change logs created on or after this timestamp.
	StartsAt *time.Time `query:"starts_at"`
	// Restricts results to change logs created on or before this timestamp.
	EndsAt *time.Time `query:"ends_at"`
}

// Returns a paginated list of inventory change logs, newest first.
//
// Filters combine with AND, while the values within a single filter combine with OR.
type ListInventoryChangeLogsEndpoint struct{}

func (e *ListInventoryChangeLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInventoryChangeLogsRequest, *apiresource.List[apiresource.InventoryChangeLog]] {
	return (&apiendpoint.APIEndpoint[*ListInventoryChangeLogsRequest, *apiresource.List[apiresource.InventoryChangeLog]]{
		Title:             "List Inventory Change Logs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventory-change-logs",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeInventoryChangeLog,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInventoryLogs, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInventoryChangeLogsRequest) (*apiresource.List[apiresource.InventoryChangeLog], *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).ListInventoryChangeLogs
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryChangeLog,
			Fields:     []string{"item", "responsible_user", "responsible_scanning_station"},
		}),
	})
}
