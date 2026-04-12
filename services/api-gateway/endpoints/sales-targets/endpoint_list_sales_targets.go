package salestargetep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListSalesTargetsRequest is the request to list sales targets for an account user.
type ListSalesTargetsRequest struct {
	// The user ID (sales rep) whose targets to list.
	SalesRepID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
}

type ListSalesTargetsEndpoint struct{}

func (e *ListSalesTargetsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesTargetsRequest, *apiresource.List[apiresource.SalesTarget]] {
	return &apiendpoint.APIEndpoint[*ListSalesTargetsRequest, *apiresource.List[apiresource.SalesTarget]]{
		Title:             "List Sales Targets",
		Description:       "Returns a paginated list of sales targets for a specific account user.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-users/{id}/sales-targets",
		Request:           &ListSalesTargetsRequest{},
		Response:          &apiresource.List[apiresource.SalesTarget]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesTargetsRequest) (*apiresource.List[apiresource.SalesTarget], *apierror.APIError) {
			return svc.(SalesTargetSvc).ListSalesTargets
		},
	}
}
