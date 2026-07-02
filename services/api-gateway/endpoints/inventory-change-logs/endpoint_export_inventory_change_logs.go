package inventorychangelogep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// ExportInventoryChangeLogsRequest is the request to export inventory change logs.
type ExportInventoryChangeLogsRequest struct {
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

// Exports inventory change logs matching the provided filters as an Excel file.
//
// Unlike the list endpoint, results are not paginated — every matching change log is included in the download.
type ExportInventoryChangeLogsEndpoint struct{}

func (e *ExportInventoryChangeLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportInventoryChangeLogsRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportInventoryChangeLogsRequest, *httptransport.FileDownload]{
		Title:             "Export Inventory Change Logs",
		Method:            http.MethodGet,
		ContentType:       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:             "/v1/operations/inventory-change-logs/actions/export",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInventoryLogs, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportInventoryChangeLogsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).ExportInventoryChangeLogs
		},
	})
}
