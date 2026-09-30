package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list settlements.
type ListSettlementsRequest struct {
	apiresource.PaginationRequest
	// Only return settlements that allocate at least one of these transactions.
	TransactionIDs []string `query:"transaction_ids"`
	// Only return settlements that allocate to at least one of these invoices.
	InvoiceIDs []string `query:"invoice_ids"`
	// Only return settlements created on or after the start of this date (`YYYY-MM-DD`, UTC). A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	StartDate *string `query:"starts_at"`
	// Only return settlements created on or before this date (`YYYY-MM-DD`, UTC), covering that whole day. A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	EndDate *string `query:"ends_at"`
}

// TODO: stop returning SettlementSummary; return the full Settlement apiresource and use proper includes values to control expansion.

// Returns a paginated list of settlements, newest first.
//
// Each entry is a condensed view that summarizes the settlement's allocations as totals per transaction type instead of listing them; retrieve a settlement to see its individual allocations. Filtering by `transaction_ids` or `invoice_ids` selects settlements with a matching allocation (one allocation must satisfy both when both are given); each entry still summarizes all of the settlement's allocations. Totals that come to zero are null.
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
