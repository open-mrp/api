package inventorychangelogep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apierror "github.com/augno/api/shared/errors"
)

// ExportInventoryChangeLogsRequest is the request to export inventory change logs with optional filters.
type ExportInventoryChangeLogsRequest struct {
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

type ExportInventoryChangeLogsEndpoint struct{}

func (e *ExportInventoryChangeLogsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportInventoryChangeLogsRequest, *httptransport.FileDownload] {
	return &apiendpoint.APIEndpoint[*ExportInventoryChangeLogsRequest, *httptransport.FileDownload]{
		Title:             "Export Inventory Change Logs",
		Description:       "Exports inventory change logs matching the provided filters as an Excel file.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/inventory-change-logs/actions/export",
		Request:           &ExportInventoryChangeLogsRequest{},
		Response:          &httptransport.FileDownload{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportInventoryChangeLogsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).ExportInventoryChangeLogs
		},
	}
}
