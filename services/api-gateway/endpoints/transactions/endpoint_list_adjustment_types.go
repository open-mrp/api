package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list adjustment types.
type ListAdjustmentTypesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of adjustment types.
type ListAdjustmentTypesEndpoint struct{}

func (e *ListAdjustmentTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAdjustmentTypesRequest, *apiresource.List[apiresource.AdjustmentType]] {
	return (&apiendpoint.APIEndpoint[*ListAdjustmentTypesRequest, *apiresource.List[apiresource.AdjustmentType]]{
		Title:             "List Adjustment Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/adjustment-types",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAdjustmentTypesRequest) (*apiresource.List[apiresource.AdjustmentType], *apierror.APIError) {
			return svc.(TransactionSvc).ListAdjustmentTypes
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAdjustmentType,
			Fields:     []string{"owner"},
		}),
	})
}
