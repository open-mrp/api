package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list open credit transactions.
type ListOpenCreditsRequest struct {
	apiresource.PaginationRequest
	// Filter by start date (inclusive, YYYY-MM-DD).
	StartDate *string `query:"start_date"`
	// Filter by end date (exclusive, YYYY-MM-DD).
	EndDate *string `query:"end_date"`
	// Filter by customer account IDs.
	CustomerIDs []string `query:"customer_ids"`
}

// Returns a paginated list of open credit transactions for the current account.
type ListOpenCreditsEndpoint struct{}

func (e *ListOpenCreditsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOpenCreditsRequest, *apiresource.List[apiresource.OpenCreditEntry]] {
	return (&apiendpoint.APIEndpoint[*ListOpenCreditsRequest, *apiresource.List[apiresource.OpenCreditEntry]]{
		Title:             "List Open Credits",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/open-credits",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOpenCreditsRequest) (*apiresource.List[apiresource.OpenCreditEntry], *apierror.APIError) {
			return svc.(TransactionAllocationSvc).ListOpenCredits
		},
	})
}
