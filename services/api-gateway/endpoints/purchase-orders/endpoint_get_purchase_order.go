package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a purchase order by ID.
type GetPurchaseOrderRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// Fields to include in the response.
	Includes []string `query:"include"`
}

type GetPurchaseOrderEndpoint struct{}

func (e *GetPurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPurchaseOrderRequest, *apiresource.PurchaseOrderDetail] {
	return &apiendpoint.APIEndpoint[*GetPurchaseOrderRequest, *apiresource.PurchaseOrderDetail]{
		Title:             "Get Purchase Order",
		Description:       "Returns a purchase order by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}",
		Request:           &GetPurchaseOrderRequest{},
		Response:          &apiresource.PurchaseOrderDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).GetPurchaseOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	}
}
