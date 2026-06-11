package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list settlements.
type ListSettlementsRequest struct {
	apiresource.PaginationRequest
	// Filter by transaction IDs present in allocations.
	TransactionIDs []string `query:"transaction_ids"`
	// Filter by invoice IDs present in allocations.
	InvoiceIDs []string `query:"invoice_ids"`
	// Only return settlements created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"start_date"`
	// Only return settlements created before this date (`YYYY-MM-DD`).
	EndDate *string `query:"end_date"`
}

// TODO: stop returning SettlementSummary; return the full Settlement apiresource and use proper includes values to control expansion.

// Returns a paginated list of settlements for the current account.
type ListSettlementsEndpoint struct{}

func (e *ListSettlementsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSettlementsRequest, *apiresource.List[apiresource.SettlementSummary]] {
	return (&apiendpoint.APIEndpoint[*ListSettlementsRequest, *apiresource.List[apiresource.SettlementSummary]]{
		Title:             "List Settlements",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/settlements",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSettlementSummary,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSettlementsRequest) (*apiresource.List[apiresource.SettlementSummary], *apierror.APIError) {
			return svc.(SettlementSvc).ListSettlements
		},
	})
}
