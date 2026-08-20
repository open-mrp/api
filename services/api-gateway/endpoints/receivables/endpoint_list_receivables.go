package receivableep

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

// Request to list all receivable entries.
type ListReceivablesRequest struct {
	apiresource.PaginationRequest
	// Compute receivable balances as of this timestamp.
	//
	// Only invoices created before the cutoff are included, only payments whose funds had landed by then are subtracted from each remaining balance, and entries already settled by the cutoff drop out. When omitted, current balances are returned for every unpaid invoice.
	CutoffDate *time.Time `query:"cutoff_at"`
}

// Returns a paginated list of outstanding receivable entries for the current account, newest invoice first.
//
// One entry is returned per invoice that is not marked paid in full, across every customer. A free-text search term (`q`) is matched against the invoice number and the customer name.
type ListReceivablesEndpoint struct{}

func (e *ListReceivablesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListReceivablesRequest, *apiresource.List[apiresource.ReceivableEntry]] {
	return (&apiendpoint.APIEndpoint[*ListReceivablesRequest, *apiresource.List[apiresource.ReceivableEntry]]{
		Title:             "List Receivables",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/receivables",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeReceivableEntry,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListReceivablesRequest) (*apiresource.List[apiresource.ReceivableEntry], *apierror.APIError) {
			return svc.(ReceivableSvc).ListReceivables
		},
	})
}
