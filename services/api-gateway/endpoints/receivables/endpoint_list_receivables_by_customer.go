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

// Request to list receivable entries for a specific customer.
type ListReceivablesByCustomerRequest struct {
	apiresource.PaginationRequest
	// Customer account ID.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// Compute receivable balances as of this timestamp.
	//
	// Only invoices created before the cutoff are included, and only allocations made before the cutoff are subtracted from each remaining balance. When omitted, current balances are returned.
	CutoffDate *time.Time `query:"cutoff_date"`
}

// Returns a paginated list of outstanding receivable entries for a specific customer account.
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
