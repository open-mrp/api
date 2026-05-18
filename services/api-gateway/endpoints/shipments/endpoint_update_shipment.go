package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a shipment.
type UpdateShipmentRequest struct {
	// Shipment ID.
	ShipmentID string `path:"id" validate:"required"`
	// Note for the shipment.
	Note *string `json:"note,omitempty" nullable:"false"`
	// Shipment number.
	Number *string `json:"number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Master tracking number.
	MasterTrackingNumber *string `json:"master_tracking_number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Carrier ID.
	CarrierID *string `json:"carrier_id,omitempty" nullable:"false" validate:"omitempty"`
	// Service level ID.
	ServiceLevelID *string `json:"service_level_id,omitempty" nullable:"true" validate:"omitempty"`
}

var sampleUpdateNote = "Updated shipping note"

var sampleUpdateShipmentRequest = &UpdateShipmentRequest{
	Note: &sampleUpdateNote,
}

func (*UpdateShipmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShipmentRequest)
}

// Partially updates a shipment.
type UpdateShipmentEndpoint struct{}

func (e *UpdateShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShipmentRequest, *apiresource.ShipmentDetail] {
	return (&apiendpoint.APIEndpoint[*UpdateShipmentRequest, *apiresource.ShipmentDetail]{
		Title:             "Update Shipment",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/shipments/{id}",
		ContentType:       "application/json",
		Request:           &UpdateShipmentRequest{},
		Response:          &apiresource.ShipmentDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
			return svc.(ShipmentSvc).UpdateShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "sales_order", "customer", "carrier", "service_level", "shipping_address", "shipped_by", "invoice", "pick"},
		}),
	}).WithDocSource(e)
}
