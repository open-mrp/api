package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
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

type ListPicksEndpoint struct{}

func (e *ListPicksEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPicksRequest, *apiresource.List[apiresource.PickSummary]] {
	return &apiendpoint.APIEndpoint[*ListPicksRequest, *apiresource.List[apiresource.PickSummary]]{
		Title:             "List Picks",
		Description:       "Returns a paginated list of picks.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/picks",
		Request:           &ListPicksRequest{},
		Response:          &apiresource.List[apiresource.PickSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.PickSummary], *apierror.APIError) {
			return svc.(PickSvc).ListPicks
		},
	}
}
