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

// Request to create a line on a sales order.
type CreateSalesOrderLineRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// ID of the product being ordered.
	ProductID string `json:"product_id" validate:"required"`
	// The product SKU recorded on the line.
	ProductSKU string `json:"product_sku" validate:"required,max=255"`
	// The product description recorded on the line.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// Quantity ordered.
	//
	// The unit must belong to the product's unit group.
	Quantity apirequest.QuantityInput `json:"quantity" validate:"required"`
	// Unit price override.
	//
	// When omitted, the line is priced server-side from the product's pricing rules (customer price, unit-conversion and volume discounts, account-price overrides) — the same pricing applied when the order is created. An explicit value is honored only for internal users. The unit cost is always resolved from the product and never taken from the request.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
}

var sampleCreateSalesOrderLineRequest = &CreateSalesOrderLineRequest{
	ProductID:  apiresource.SampleProductID,
	ProductSKU: "WIDGET-001",
	Quantity:   apirequest.QuantityInput{Value: "10", UnitID: apiresource.SampleUnitID},
}

func (*CreateSalesOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSalesOrderLineRequest)
}

// Adds a line item to a sales order.
//
// The new line is appended below the existing product lines, keeping the order's freight and discount lines at the bottom. When the order has already been issued, the line is added to its pick as outstanding work and the pick is reopened if it had been finished.
type CreateSalesOrderLineEndpoint struct{}

func (e *CreateSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesOrderLineRequest, *apiresource.SalesOrderLine] {
	return (&apiendpoint.APIEndpoint[*CreateSalesOrderLineRequest, *apiresource.SalesOrderLine]{
		Title:             "Create Sales Order Line",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrderLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrderLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrderLine,
			Fields:     []string{"product", "quantity_ordered", "unit_price", "unit_cost", "totals"},
		}),
	})
}
