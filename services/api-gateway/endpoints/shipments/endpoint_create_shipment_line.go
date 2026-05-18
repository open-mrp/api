package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a shipment line.
type CreateShipmentLineRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// Sales order line ID.
	SalesOrderLineID string `json:"sales_order_line_id" validate:"required"`
	// Quantity value.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// Quantity unit ID.
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

// Creates a line on a shipment.
type CreateShipmentLineEndpoint struct{}

func (e *CreateShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateShipmentLineRequest, *apiresource.ShipmentLine] {
	return (&apiendpoint.APIEndpoint[*CreateShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:             "Create Shipment Line",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/{shipment_id}/lines",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).CreateShipmentLine
		},
	})
}
