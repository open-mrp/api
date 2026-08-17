package salestargetep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list sales targets for an account user.
type ListSalesTargetsRequest struct {
	// The account user (sales rep) whose targets to list.
	//
	// Must be an active account user in your account.
	SalesRepID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
}

// Returns the revenue goals set for one sales rep, most recent period first.
//
// This endpoint does not support cursor pagination; passing a `cursor` returns a validation error, and the response carries no page cursors. Requesting targets for someone who is not an active account user in your account returns a not-found error.
//
// Pass `q` to narrow the list to targets whose ID or goal amount contains the search text.
type ListSalesTargetsEndpoint struct{}

func (e *ListSalesTargetsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesTargetsRequest, *apiresource.List[apiresource.SalesTarget]] {
	return (&apiendpoint.APIEndpoint[*ListSalesTargetsRequest, *apiresource.List[apiresource.SalesTarget]]{
		Title:             "List Sales Targets",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-users/{id}/sales-targets",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesTarget,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSalesTargets, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesTargetsRequest) (*apiresource.List[apiresource.SalesTarget], *apierror.APIError) {
			return svc.(SalesTargetSvc).ListSalesTargets
		},
	})
}
