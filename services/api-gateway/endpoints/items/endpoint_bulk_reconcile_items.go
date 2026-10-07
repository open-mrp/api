package itemep

import (
	"context"
	"net/http"

	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// One item to reconcile in a bulk reconcile request.
type BulkReconcileItemInput struct {
	// SKU of the item to reconcile.
	//
	// Items whose SKU does not match an existing item are reported in the response's `skipped_items` rather than failing the request.
	SKU string `json:"sku" validate:"required"`
	// Abbreviation of the unit `quantity` is counted in (e.g. `kg`), matched without regard to case.
	//
	// It must be the item's base unit or another unit in its category's unit group; the quantity is converted from it into the base unit before it is applied, so `2 dz` against an item stocked in eaches reconciles 24. A row whose abbreviation matches no unit, or a unit outside the item's unit group, is reported in the response's `errors` and writes nothing.
	Unit string `json:"unit" validate:"required"`
	// Quantity to apply, interpreted according to the request's `reconcile_type`.
	//
	// A decimal string rather than a number: a quantity that has been through a binary float is not the quantity you sent.
	Quantity string `json:"quantity" validate:"required" format:"decimal"`
}

// Request to reconcile inventory for many items at once.
type BulkReconcileItemsRequest struct {
	// Items to reconcile, at most 1,000 rows per request. Split a larger count across requests.
	Data []BulkReconcileItemInput `json:"data" validate:"required,max=1000"`
	// How each item's quantity is applied to its current quantity.
	//
	// - `addition`: adds the quantity to the item's current quantity.
	// - `force`: sets the item's current quantity to exactly the given quantity.
	ReconcileType constants.ItemReconcileType `json:"reconcile_type" validate:"required"`
}

var sampleBulkReconcileItemsRequest = &BulkReconcileItemsRequest{
	Data: []BulkReconcileItemInput{
		{
			SKU:      apiresource.SampleItemSKU,
			Unit:     apiresource.SampleUnitAbbreviation,
			Quantity: "10.5",
		},
	},
	ReconcileType: constants.ItemReconcileTypeAddition,
}

func (*BulkReconcileItemsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkReconcileItemsRequest)
}

// Reconciles inventory for multiple items by SKU in one call, the bulk equivalent of counting stock and correcting the books.
//
// `reconcile_type` controls whether each quantity is added to the item's current quantity (`addition`) or replaces it (`force`). The figure a `force` measures against is what is on hand net of demand nothing has covered, the same basis the single-item endpoint uses. Each quantity is converted from its row's unit into the item's base unit, and `previous_quantity` and `new_quantity` are reported in that base unit. A SKU listed twice applies each row in turn.
//
// The response reports each row as reconciled, skipped (unknown SKU), or errored (unknown unit, or a unit outside the item's unit group), so a problem with one row does not fail the rest. Rows are written in batches of 50, each in its own transaction; a batch that cannot be written is rolled back whole and every row in it is reported in `errors`, while the batches before and after it still apply. Resubmit only the errored rows — in `addition` mode, resubmitting the whole request would apply the reconciled rows twice.
//
// At most 1,000 rows per request, and a request body of at most 8 MB. Each correction is written to the item's inventory audit trail as a user correction, attributed to the caller.
type BulkReconcileItemsEndpoint struct{}

func (e *BulkReconcileItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkReconcileItemsRequest, *apiresource.BulkReconcileItemsResponse] {
	return (&apiendpoint.APIEndpoint[*BulkReconcileItemsRequest, *apiresource.BulkReconcileItemsResponse]{
		Title:               "Bulk Reconcile Items",
		Method:              http.MethodPost,
		Route:               "/v1/catalog/items/actions/bulk-reconcile",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionCreate}},
		Extras:              apiendpoint.APIEndpointExtras{MaxJSONBodyBytes: httptransport.MaxJSONBodyBytes},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkReconcileItemsRequest) (*apiresource.BulkReconcileItemsResponse, *apierror.APIError) {
			return svc.(ItemSvc).BulkReconcileItems
		},
	})
}
