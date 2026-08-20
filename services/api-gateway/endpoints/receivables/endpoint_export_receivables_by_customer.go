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
	// ID of the customer account whose outstanding balances are exported.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// Compute receivable balances as of this timestamp.
	//
	// Only invoices created before the cutoff are included, only payments whose funds had landed by then are subtracted from each remaining balance, and entries already settled by the cutoff drop out. When omitted, current balances are returned for every unpaid invoice.
	CutoffDate *time.Time `query:"cutoff_at"`
}

// Exports a single customer's outstanding receivable entries as a downloadable CSV file.
//
// The response is the file itself rather than a JSON resource, and it covers every open invoice for the customer instead of one page of results. When a cutoff date is supplied, it is included in the generated file name.
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
