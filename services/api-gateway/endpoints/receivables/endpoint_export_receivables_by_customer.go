package receivableep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to export receivable entries for a specific customer as CSV.
type ExportReceivablesByCustomerRequest struct {
	// Customer account ID.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// Compute receivable balances as of this timestamp.
	//
	// Only invoices created before the cutoff are included, and only allocations made before the cutoff are subtracted from each remaining balance. When omitted, current balances are returned.
	CutoffDate *time.Time `query:"cutoff_date"`
}

// Exports all outstanding receivable entries for a specific customer account as a CSV file.
type ExportReceivablesByCustomerEndpoint struct{}

func (e *ExportReceivablesByCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportReceivablesByCustomerRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportReceivablesByCustomerRequest, *httptransport.FileDownload]{
		Title:             "Export Receivables by Customer",
		Method:            http.MethodGet,
		ContentType:       "text/csv",
		Route:             "/v1/finance/receivables/accounts/{account_id}/actions/export",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportReceivablesByCustomerRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(ReceivableSvc).ExportReceivablesByCustomer
		},
	})
}
