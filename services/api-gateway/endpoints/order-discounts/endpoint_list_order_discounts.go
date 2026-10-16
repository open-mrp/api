package orderdiscountep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list order discounts.
type ListOrderDiscountsRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `name` or `code`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return discounts with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
}

// Returns a list of order discounts.
//
// Returns your account's order discounts, newest first.
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
		ObjectType:        constants.ObjectTypeOrderDiscount,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOrderDiscountsRequest) (*apiresource.List[apiresource.OrderDiscount], *apierror.APIError) {
			return svc.(OrderDiscountSvc).ListOrderDiscounts
		},
	})
}
