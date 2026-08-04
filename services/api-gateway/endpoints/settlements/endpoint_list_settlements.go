package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list settlements.
type ListSettlementsRequest struct {
	apiresource.PaginationRequest
	// Only return settlements that allocate at least one of these transactions.
	TransactionIDs []string `query:"transaction_ids"`
	// Only return settlements that allocate to at least one of these invoices.
	InvoiceIDs []string `query:"invoice_ids"`
	// Only return settlements created on or after the start of this date (`YYYY-MM-DD`, UTC).
	StartDate *string `query:"start_date"`
	// Only return settlements created up to the start of this date (`YYYY-MM-DD`, UTC).
	//
	// Settlements created later on that day are excluded, so pass the following day to cover a full day.
	EndDate *string `query:"end_date"`
}

// TODO: stop returning SettlementSummary; return the full Settlement apiresource and use proper includes values to control expansion.

// Returns a paginated list of settlements, newest first.
//
// Each entry is a condensed view that summarizes the settlement's allocations as totals per transaction type instead of listing them; retrieve a settlement to see its individual allocations. Filtering by `transaction_ids` or `invoice_ids` also narrows each entry's aggregates to just the matching allocations, and when both are supplied a settlement matches only if one of its allocations satisfies both.
type ListSettlementsEndpoint struct{}

func (e *ListSettlementsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSettlementsRequest, *apiresource.List[apiresource.SettlementSummary]] {
	return (&apiendpoint.APIEndpoint[*ListSettlementsRequest, *apiresource.List[apiresource.SettlementSummary]]{
		Title:               "List Settlements",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/finance/settlements",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSettlements, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeSettlementSummary,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSettlementsRequest) (*apiresource.List[apiresource.SettlementSummary], *apierror.APIError) {
			return svc.(SettlementSvc).ListSettlements
		},
	})
}
