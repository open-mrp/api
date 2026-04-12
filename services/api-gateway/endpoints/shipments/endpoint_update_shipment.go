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

// UpdateShipmentRequest is the request to partially update a shipment.
type UpdateShipmentRequest struct {
	// The ID of the shipment to update.
	ShipmentID string `path:"id" validate:"required"`
	// An optional note for the shipment.
	Note *string `json:"note,omitempty" nullable:"false"`
	// The shipment number.
	Number *string `json:"number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The master tracking number for the shipment.
	MasterTrackingNumber *string `json:"master_tracking_number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The ID of the carrier for this shipment.
	CarrierID *string `json:"carrier_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// The ID of the service level for this shipment.
	ServiceLevelID *string `json:"service_level_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
}

var sampleUpdateNote = "Updated shipping note"

var sampleUpdateShipmentRequest = &UpdateShipmentRequest{
	Note: &sampleUpdateNote,
}

func (*UpdateShipmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShipmentRequest)
}

type UpdateShipmentEndpoint struct{}

func (e *UpdateShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShipmentRequest, *apiresource.ShipmentDetail] {
	return &apiendpoint.APIEndpoint[*UpdateShipmentRequest, *apiresource.ShipmentDetail]{
		Title:             "Update Shipment",
		Description:       "Partially updates a shipment.",
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
	}
}
