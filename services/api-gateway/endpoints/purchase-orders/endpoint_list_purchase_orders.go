package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list purchase orders.
type ListPurchaseOrdersRequest struct {
	apiresource.PaginationRequest
	// Filter to orders with any of these statuses (`estimate`, `issued`, `fulfilled`).
	StatusCodes []string `query:"status_codes"`
	// Filter to orders with at least one line referencing any of these items.
	ItemIDs []string `query:"item_ids"`
	// Filter to orders placed with any of these suppliers.
	SupplierIDs []string `query:"supplier_ids"`
	// Filter to orders created on or after this date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter to orders created on or before this date (inclusive).
	EndDate *string `query:"end_date"`
}

// Returns a paginated list of purchase orders for the current account.
type ListPurchaseOrdersEndpoint struct{}

func (e *ListPurchaseOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPurchaseOrdersRequest, *apiresource.List[apiresource.PurchaseOrder]] {
	return (&apiendpoint.APIEndpoint[*ListPurchaseOrdersRequest, *apiresource.List[apiresource.PurchaseOrder]]{
		Title:             "List Purchase Orders",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPurchaseOrdersRequest) (*apiresource.List[apiresource.PurchaseOrder], *apierror.APIError) {
			return svc.(PurchaseOrderSvc).ListPurchaseOrders
		},
		ObjectType: constants.ObjectTypePurchaseOrder,
		// The list summary stashes the supplier inline (cross-account, like the
		// receiving-order supplier); expose it so list rows can request it.
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "lines"},
		}),
	})
}
