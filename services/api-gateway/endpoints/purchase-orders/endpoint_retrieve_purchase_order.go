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
type RetrievePurchaseOrderRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
}

// Returns a purchase order by ID.
type RetrievePurchaseOrderEndpoint struct{}

func (e *RetrievePurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePurchaseOrderRequest, *apiresource.PurchaseOrderDetail] {
	return (&apiendpoint.APIEndpoint[*RetrievePurchaseOrderRequest, *apiresource.PurchaseOrderDetail]{
		Title:             "Retrieve Purchase Order",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePurchaseOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).GetPurchaseOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	})
}
