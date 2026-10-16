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

// Request to create an order discount.
type CreateOrderDiscountRequest struct {
	// Display name of the discount, such as `Spring promotion`.
	Name string `json:"name" validate:"required,max=255"`
	// The code a buyer enters. Unique within your account, matched case-insensitively, so `SAVE10` collides with `save10`.
	Code string `json:"code" validate:"required,max=255"`
	// How much comes off: `{ "type": "percent_off", "percent_off": "10" }` for 10% off the order total, or `{ "type": "amount_off", "amount_off": { "amount": "25.00", "currency": "usd" } }`.
	Discount apirequest.DiscountInput `json:"discount" validate:"required"`
	// Whether the code can be applied to new orders. Defaults to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleCreateOrderDiscountRequest = &CreateOrderDiscountRequest{
	Name:     "10% Off",
	Code:     "SAVE10",
	Discount: apirequest.DiscountInput{Type: constants.DiscountTypePercentOff, PercentOff: field.Some("10")},
}

func (*CreateOrderDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateOrderDiscountRequest)
}

// Creates an order discount.
//
// Buyers apply it to an order by entering its code. Reusing a code another discount holds returns 409 `resource_conflict`.
type CreateOrderDiscountEndpoint struct{}

func (e *CreateOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateOrderDiscountRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*CreateOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Create Order Discount",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/order-discounts",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionCreate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).CreateOrderDiscount
		},
		LocationFunc: func(resp *apiresource.OrderDiscount) string {
			return "/v1/sales/order-discounts/" + resp.ID
		},
	})
}
