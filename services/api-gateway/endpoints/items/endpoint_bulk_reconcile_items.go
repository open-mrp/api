package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// BulkReconcileItemInput is the input for a single item in a bulk reconcile operation.
type BulkReconcileItemInput struct {
	// SKU of the item to reconcile.
	//
	// Items whose SKU does not match an existing item are reported in the response's `skipped_items` rather than failing the request.
	SKU string `json:"sku" validate:"required"`
	// Abbreviation of the unit the quantity is expressed in (e.g. `kg`).
	//
	// Must match a unit defined on the account; items with an unknown unit are reported in the response's `errors`.
	Unit string `json:"unit" validate:"required"`
	// Quantity to apply, interpreted according to the request's `reconcile_type`.
	Quantity float64 `json:"quantity" validate:"required"`
}

// BulkReconcileItemsRequest is the request to bulk reconcile item inventory.
type BulkReconcileItemsRequest struct {
	// Items to reconcile.
	Data []BulkReconcileItemInput `json:"data" validate:"required"`
	// How each item's quantity is applied to its current on-hand inventory.
	//
	// - `addition`: adds the quantity to the item's current on-hand quantity.
	// - `force`: sets the item's on-hand quantity to exactly the given quantity.
	ReconcileType string `json:"reconcile_type" validate:"required"`
}

var sampleBulkReconcileItemsRequest = &BulkReconcileItemsRequest{
	Data: []BulkReconcileItemInput{
		{
			SKU:      apiresource.SampleItemSKU,
			Unit:     apiresource.SampleUnitAbbreviation,
			Quantity: 10.5,
		},
	},
	ReconcileType: "addition",
}

func (*BulkReconcileItemsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkReconcileItemsRequest)
}

// Reconciles on-hand inventory for multiple items by SKU in one call.
//
// `reconcile_type` controls whether each quantity is added to the item's current on-hand quantity (`addition`) or replaces it (`force`). The response reports each item as reconciled, skipped (e.g. unknown SKU), or errored (e.g. unknown unit), so a problem with one item does not fail the rest of the batch.
type BulkReconcileItemsEndpoint struct{}

func (e *BulkReconcileItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkReconcileItemsRequest, *apiresource.BulkReconcileItemsResponse] {
	return (&apiendpoint.APIEndpoint[*BulkReconcileItemsRequest, *apiresource.BulkReconcileItemsResponse]{
		Title:               "Bulk Reconcile Items",
		Method:              http.MethodPost,
		Route:               "/v1/catalog/items/actions/bulk-reconcile",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkReconcileItemsRequest) (*apiresource.BulkReconcileItemsResponse, *apierror.APIError) {
			return svc.(ItemSvc).BulkReconcileItems
		},
	})
}
