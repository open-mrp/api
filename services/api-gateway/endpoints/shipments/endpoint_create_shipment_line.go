package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a shipment line.
type CreateShipmentLineRequest struct {
	// ID of the shipment to add the line to.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// ID of the sales order line this shipment line fulfills.
	SalesOrderLineID string `json:"sales_order_line_id" validate:"required"`
	// Quantity shipped, as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// ID of the unit of measure for `quantity_value`.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
}

var sampleCreateShipmentLineRequest = &CreateShipmentLineRequest{
	SalesOrderLineID: apiresource.SampleSalesOrderLineID,
	QuantityValue:    "10.000000000000000000000000000000",
	QuantityUnitID:   apiresource.SampleUnitID,
}

func (*CreateShipmentLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateShipmentLineRequest)
}

// Adds a line to a shipment, recording how much of a sales order line the shipment carries.
//
// The line only records what the shipment carries: it does not touch the pick for the order, so the pick's lines keep their existing packed state.
type CreateShipmentLineEndpoint struct{}

func (e *CreateShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateShipmentLineRequest, *apiresource.ShipmentLine] {
	return (&apiendpoint.APIEndpoint[*CreateShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:               "Create Shipment Line",
		Method:              http.MethodPost,
		Route:               "/v1/operations/shipments/{shipment_id}/lines",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).CreateShipmentLine
		},
	})
}
