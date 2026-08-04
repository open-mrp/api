package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list open credit transactions.
type ListOpenCreditsRequest struct {
	apiresource.PaginationRequest
	// Only include transactions created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"start_date"`
	// Only include transactions created before this date (`YYYY-MM-DD`).
	EndDate *string `query:"end_date"`
	// Filter by customer account IDs.
	CustomerIDs []string `query:"customer_ids"`
}

// Returns a paginated list of customer transactions that still have money left to apply to invoices, newest first.
//
// Membership is driven by each transaction's `is_fully_allocated` flag rather than by a recomputed balance, so a transaction remains listed until that flag is set. Free-text search matches the transaction ID, transaction number, customer name, and note.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSettlements, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeOpenCreditEntry,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOpenCreditsRequest) (*apiresource.List[apiresource.OpenCreditEntry], *apierror.APIError) {
			return svc.(TransactionAllocationSvc).ListOpenCredits
		},
	})
}
