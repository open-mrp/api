package receivableep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListReceivablesByCustomerRequest is the request to list receivable entries for a specific customer.
type ListReceivablesByCustomerRequest struct {
	apiresource.PaginationRequest
	// The customer account ID.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// Optional cutoff date for the receivables snapshot.
	CutoffDate *time.Time `query:"cutoff_date"`
}

type ListReceivablesByCustomerEndpoint struct{}

func (e *ListReceivablesByCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListReceivablesByCustomerRequest, *apiresource.List[apiresource.ReceivableEntry]] {
	return &apiendpoint.APIEndpoint[*ListReceivablesByCustomerRequest, *apiresource.List[apiresource.ReceivableEntry]]{
		Title:             "List Receivables by Customer",
		Description:       "Returns a paginated list of receivable entries for a specific customer account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/receivables/accounts/{account_id}",
		Request:           &ListReceivablesByCustomerRequest{},
		Response:          &apiresource.List[apiresource.ReceivableEntry]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListReceivablesByCustomerRequest) (*apiresource.List[apiresource.ReceivableEntry], *apierror.APIError) {
			return svc.(ReceivableSvc).ListReceivablesByCustomer
		},
	}
}
