package receivableep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apierror "github.com/augno/api/shared/errors"
)

// Request to export receivable entries for a specific customer as CSV.
type ExportReceivablesByCustomerRequest struct {
	// Customer account ID.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// Cutoff date for the receivables snapshot.
	CutoffDate *time.Time `query:"cutoff_date"`
}

type ExportReceivablesByCustomerEndpoint struct{}

func (e *ExportReceivablesByCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportReceivablesByCustomerRequest, *httptransport.FileDownload] {
	return &apiendpoint.APIEndpoint[*ExportReceivablesByCustomerRequest, *httptransport.FileDownload]{
		Title:             "Export Receivables by Customer",
		Description:       "Exports all receivable entries for a specific customer account as a CSV file.",
		Method:            http.MethodGet,
		ContentType:       "text/csv",
		Route:             "/v1/finance/receivables/accounts/{account_id}/actions/export",
		Request:           &ExportReceivablesByCustomerRequest{},
		Response:          &httptransport.FileDownload{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportReceivablesByCustomerRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(ReceivableSvc).ExportReceivablesByCustomer
		},
	}
}
