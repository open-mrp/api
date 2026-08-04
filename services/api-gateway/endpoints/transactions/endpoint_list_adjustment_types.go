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

// Returns a paginated list of the adjustment categories that can be recorded on an adjustment transaction, such as discounts, fees, and write-offs.
//
// Adjustment types are platform-provided and identical for every account. Free-text search matches the display name.
type ListAdjustmentTypesEndpoint struct{}

func (e *ListAdjustmentTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAdjustmentTypesRequest, *apiresource.List[apiresource.AdjustmentType]] {
	return (&apiendpoint.APIEndpoint[*ListAdjustmentTypesRequest, *apiresource.List[apiresource.AdjustmentType]]{
		Title:             "List Adjustment Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/adjustment-types",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAdjustmentTypesRequest) (*apiresource.List[apiresource.AdjustmentType], *apierror.APIError) {
			return svc.(TransactionSvc).ListAdjustmentTypes
		},
		ObjectType: constants.ObjectTypeAdjustmentType,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAdjustmentType,
			Fields:     []string{"owner"},
		}),
	})
}
