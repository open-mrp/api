package orderdiscountep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update an order discount.
type UpdateOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
	// Display name of the discount.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// The code a buyer enters. Unique within your account, matched case-insensitively.
	Code field.Optional[string] `json:"code,omitzero" validate:"omitempty,max=255"`
	// How much comes off. Replaces the whole discount.
	Discount field.Optional[apirequest.DiscountInput] `json:"discount,omitzero"`
	// Whether the code can be applied to new orders.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleUpdateOrderDiscountRequest = &UpdateOrderDiscountRequest{
	Name:   field.Some("15% Off"),
	Status: field.Some(constants.ArchiveStatusArchived),
}

func (*UpdateOrderDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateOrderDiscountRequest)
}

// Updates an order discount.
//
// Only fields you send are changed. Orders that already used the code keep their discount. Archive a code by setting `status` to `archived`.
type UpdateOrderDiscountEndpoint struct{}

func (e *UpdateOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateOrderDiscountRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*UpdateOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Update Order Discount",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).UpdateOrderDiscount
		},
	})
}
