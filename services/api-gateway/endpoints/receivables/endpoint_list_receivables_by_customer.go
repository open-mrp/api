package receivableep

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

// Request to list receivable entries for a specific customer.
type ListReceivablesByCustomerRequest struct {
	apiresource.PaginationRequest
	// ID of the customer account whose outstanding balances are listed.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// Compute receivable balances as of this timestamp.
	//
	// Only invoices created before the cutoff are included, only payments whose funds had landed by then are subtracted from each remaining balance, and entries already settled by the cutoff drop out. When omitted, current balances are returned for every unpaid invoice.
	CutoffDate *time.Time `query:"cutoff_at"`
}

// Returns a paginated list of outstanding receivable entries for a single customer account, newest invoice first.
//
// One entry is returned per invoice billed to that customer that is not marked paid in full. Invoices billed to the customer's child accounts are not included.
type ListReceivablesByCustomerEndpoint struct{}

func (e *ListReceivablesByCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListReceivablesByCustomerRequest, *apiresource.List[apiresource.ReceivableEntry]] {
	return (&apiendpoint.APIEndpoint[*ListReceivablesByCustomerRequest, *apiresource.List[apiresource.ReceivableEntry]]{
		Title:             "List Receivables by Customer",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/receivables/accounts/{account_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeReceivableEntry,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListReceivablesByCustomerRequest) (*apiresource.List[apiresource.ReceivableEntry], *apierror.APIError) {
			return svc.(ReceivableSvc).ListReceivablesByCustomer
		},
	})
}
