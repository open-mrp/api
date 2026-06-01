package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type ListPicksRequest struct {
	apiresource.PaginationRequest
	Status           *string  `query:"status"`
	CustomerIDs      []string `query:"customer_ids"`
	ProductLineIDs   []string `query:"product_line_ids"`
	CustomerGroupIDs []string `query:"customer_group_ids"`
	DepartmentIDs    []string `query:"department_ids"`
	StartDate        *string  `query:"start_date"`
	EndDate          *string  `query:"end_date"`
}

// TODO: stop returning PickSummary; return the full Pick apiresource and use proper includes values to control expansion.

// Returns a paginated list of picks.
type ListPicksEndpoint struct{}

func (e *ListPicksEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPicksRequest, *apiresource.List[apiresource.PickSummary]] {
	return (&apiendpoint.APIEndpoint[*ListPicksRequest, *apiresource.List[apiresource.PickSummary]]{
		Title:             "List Picks",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePick,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.PickSummary], *apierror.APIError) {
			return svc.(PickSvc).ListPicks
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
			Fields:     []string{"sales_order"},
		}),
	})
}
