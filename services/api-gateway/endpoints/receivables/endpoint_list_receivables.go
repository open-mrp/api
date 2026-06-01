package receivableep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list all receivable entries.
type ListReceivablesRequest struct {
	apiresource.PaginationRequest
	// Cutoff date for the receivables snapshot.
	CutoffDate *time.Time `query:"cutoff_date"`
}

// Returns a paginated list of receivable entries for the current account.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListReceivablesRequest) (*apiresource.List[apiresource.ReceivableEntry], *apierror.APIError) {
			return svc.(ReceivableSvc).ListReceivables
		},
	})
}
