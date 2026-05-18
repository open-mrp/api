package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// BulkReconcileItemInput is the input for a single item in a bulk reconcile operation.
type BulkReconcileItemInput struct {
	// Item SKU.
	SKU string `json:"sku" validate:"required"`
	// Unit abbreviation for the quantity.
	Unit string `json:"unit" validate:"required"`
	// Quantity.
	Quantity float64 `json:"quantity" validate:"required"`
}

// BulkReconcileItemsRequest is the request to bulk reconcile item inventory.
type BulkReconcileItemsRequest struct {
	// Items to reconcile.
	Data []BulkReconcileItemInput `json:"data" validate:"required"`
	// Reconcile type: "addition" or "force".
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

// Reconciles inventory for multiple items by SKU, either adding to or forcing the exact quantity depending on reconcile_type.
type BulkReconcileItemsEndpoint struct{}

func (e *BulkReconcileItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkReconcileItemsRequest, *apiresource.BulkReconcileItemsResponse] {
	return (&apiendpoint.APIEndpoint[*BulkReconcileItemsRequest, *apiresource.BulkReconcileItemsResponse]{
		Title:             "Bulk Reconcile Items",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/items/actions/bulk-reconcile",
		ContentType:       "application/json",
		Request:           &BulkReconcileItemsRequest{},
		Response:          &apiresource.BulkReconcileItemsResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkReconcileItemsRequest) (*apiresource.BulkReconcileItemsResponse, *apierror.APIError) {
			return svc.(ItemSvc).BulkReconcileItems
		},
	}).WithDocSource(e)
}
