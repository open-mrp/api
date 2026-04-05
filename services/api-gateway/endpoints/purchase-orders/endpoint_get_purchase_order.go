package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetPurchaseOrderRequest is the request to retrieve a single purchase order by ID.
type GetPurchaseOrderRequest struct {
	// The ID of the purchase order to retrieve.
	PurchaseOrderID string `path:"id" validate:"required"`
	// The fields to include in the response.
	Includes []string `query:"include"`
}

type GetPurchaseOrderEndpoint struct{}

func (e *GetPurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPurchaseOrderRequest, *apiresource.PurchaseOrderDetail] {
	return &apiendpoint.APIEndpoint[*GetPurchaseOrderRequest, *apiresource.PurchaseOrderDetail]{
		Title:             "Get Purchase Order",
		Description:       "Returns a single purchase order by its ID.",
		Method:            http.MethodGet,
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
