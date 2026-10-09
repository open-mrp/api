package transactionallocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list open credit transactions.
type ListOpenCreditsRequest struct {
	apiresource.PaginationRequest
	// Only include credits whose funds were received on or after this date (`YYYY-MM-DD`, UTC). A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	StartDate *string `query:"starts_at"`
	// Only include credits whose funds were received before the end of this date (`YYYY-MM-DD`, UTC). A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	EndDate *string `query:"ends_at"`
	// Filter by customer account IDs.
	CustomerIDs []string `query:"customer_ids"`
}

// Returns a paginated list of customer transactions that still have money left to apply to invoices, most recently received first.
//
// A transaction is listed once its funds have been received, while it is not marked fully allocated and its allocations leave part of its amount unapplied. Free-text search matches the transaction ID, transaction number, customer name, and note.
type ListOpenCreditsEndpoint struct{}

func (e *ListOpenCreditsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOpenCreditsRequest, *apiresource.List[apiresource.OpenCreditEntry]] {
	return (&apiendpoint.APIEndpoint[*ListOpenCreditsRequest, *apiresource.List[apiresource.OpenCreditEntry]]{
		Title:             "List Open Credits",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/open-credits",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
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
