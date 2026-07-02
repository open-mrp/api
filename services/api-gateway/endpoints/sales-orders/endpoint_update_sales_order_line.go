package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a sales order line.
type UpdateSalesOrderLineRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Sales order line ID.
	SalesOrderLineID string `path:"line_id" validate:"required"`
	// Item ID.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// Product SKU.
	ProductSKU field.Optional[string] `json:"product_sku,omitzero" validate:"omitempty,max=255"`
	// Product description.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// Quantity ordered.
	Quantity field.Optional[apirequest.QuantityInput] `json:"quantity,omitzero" validate:"omitempty"`
	// Unit price.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero" validate:"omitempty"`
	// Unit cost.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero" validate:"omitempty"`
}

var sampleUpdateSOLineItemID = apiresource.SampleItemID
var sampleUpdateSalesOrderLineRequest = &UpdateSalesOrderLineRequest{
	ItemID: field.Some(sampleUpdateSOLineItemID),
	Quantity: field.Some(apirequest.QuantityInput{
		Value:  "20",
		UnitID: apiresource.SampleUnitID,
	}),
	UnitPrice: field.Some(apirequest.RateInput{
		Value:             "30.00",
		NumeratorUnitID:   apiresource.SampleUnitID,
		DenominatorUnitID: apiresource.SampleUnitID,
	}),
}

func (*UpdateSalesOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSalesOrderLineRequest)
}

// Partially updates a sales order line item.
type UpdateSalesOrderLineEndpoint struct{}

func (e *UpdateSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSalesOrderLineRequest, *apiresource.SalesOrderLine] {
	return (&apiendpoint.APIEndpoint[*UpdateSalesOrderLineRequest, *apiresource.SalesOrderLine]{
		Title:             "Update Sales Order Line",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrderLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError) {
			return svc.(SalesOrderSvc).UpdateSalesOrderLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrderLine,
			Fields:     []string{"product", "quantity_ordered", "unit_price", "unit_cost", "totals"},
		}),
	})
}
