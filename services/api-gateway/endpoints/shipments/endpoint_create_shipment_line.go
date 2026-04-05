package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateShipmentLineRequest is the request to create a new shipment line.
type CreateShipmentLineRequest struct {
	// The ID of the shipment to add the line to.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// The ID of the sales order line to ship.
	SalesOrderLineID string `json:"sales_order_line_id" validate:"required"`
	// The quantity value to ship.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// The ID of the unit for the quantity.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
}

var sampleCreateShipmentLineRequest = &CreateShipmentLineRequest{
	SalesOrderLineID: apiresource.SampleSalesOrderLineDetailID,
	QuantityValue:    "10.000000000000000000000000000000",
	QuantityUnitID:   apiresource.SampleUnitID,
}

func (*CreateShipmentLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateShipmentLineRequest)
}

type CreateShipmentLineEndpoint struct{}

func (e *CreateShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateShipmentLineRequest, *apiresource.ShipmentLine] {
	return &apiendpoint.APIEndpoint[*CreateShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:             "Create Shipment Line",
		Description:       "Creates a new line on a shipment.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/{shipment_id}/lines",
		ContentType:       "application/json",
		Request:           &CreateShipmentLineRequest{},
		Response:          &apiresource.ShipmentLine{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).CreateShipmentLine
		},
	}
}
