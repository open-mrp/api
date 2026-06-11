package salestargetep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list sales targets for an account user.
type ListSalesTargetsRequest struct {
	// ID of the account user (sales rep) whose targets to list.
	SalesRepID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
}

// Returns the sales targets for an account user.
//
// This endpoint does not support cursor pagination; passing a `cursor` returns a validation error.
type ListSalesTargetsEndpoint struct{}

func (e *ListSalesTargetsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesTargetsRequest, *apiresource.List[apiresource.SalesTarget]] {
	return (&apiendpoint.APIEndpoint[*ListSalesTargetsRequest, *apiresource.List[apiresource.SalesTarget]]{
		Title:             "List Sales Targets",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-users/{id}/sales-targets",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesTarget,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesTargetsRequest) (*apiresource.List[apiresource.SalesTarget], *apierror.APIError) {
			return svc.(SalesTargetSvc).ListSalesTargets
		},
	})
}
