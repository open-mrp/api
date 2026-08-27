package inventorychangelogep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to export inventory change logs.
type ExportInventoryChangeLogsRequest struct {
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

// Exports inventory change logs matching the provided filters as an Excel file.
//
// Unlike the list endpoint, results are not paginated — every matching change log is included in the download, newest first. The download is named for the date range you requested, using `all` in place of a bound you left open.
type ExportInventoryChangeLogsEndpoint struct{}

func (e *ExportInventoryChangeLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportInventoryChangeLogsRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportInventoryChangeLogsRequest, *httptransport.FileDownload]{
		Title:             "Export Inventory Change Logs",
		Method:            http.MethodGet,
		ContentType:       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:             "/v1/operations/inventory-change-logs/actions/export",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInventoryLogs, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportInventoryChangeLogsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).ExportInventoryChangeLogs
		},
	})
}
