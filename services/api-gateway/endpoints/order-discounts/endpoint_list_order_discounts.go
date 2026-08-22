package orderdiscountep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/auth-service/pkg/types"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list order discounts.
type ListOrderDiscountsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of the order discounts defined for the current account, newest first.
//
// Pass `q` to narrow the list to discounts whose name or code contains the search text.
type ListOrderDiscountsEndpoint struct{}

func (e *ListOrderDiscountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOrderDiscountsRequest, *apiresource.List[apiresource.OrderDiscount]] {
	return (&apiendpoint.APIEndpoint[*ListOrderDiscountsRequest, *apiresource.List[apiresource.OrderDiscount]]{
		Title:             "List Order Discounts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/order-discounts",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOrderDiscountsRequest) (*apiresource.List[apiresource.OrderDiscount], *apierror.APIError) {
			return svc.(OrderDiscountSvc).ListOrderDiscounts
		},
	})
}
