package shippingcaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a shipping case.
type UpdateShippingCaseRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
	// Carrier tracking number to set on the case, replacing any number already recorded.
	TrackingNumber field.Optional[string] `json:"tracking_number,omitzero" validate:"omitempty,max=255"`
	// New value for the case's freight cost, as a decimal string.
	FreightAmountValue field.Optional[string] `json:"freight_amount_value,omitzero"`
	// ID of the currency unit the case's freight cost is expressed in.
	//
	// Changing the unit relabels the stored freight cost; the number itself is never converted, so send `freight_amount_value` alongside it when the amount should change too.
	FreightAmountUnitID field.Optional[string] `json:"freight_amount_unit_id,omitzero" validate:"omitempty"`
	// New value for the case's freight weight, as a decimal string.
	FreightWeightValue field.Optional[string] `json:"freight_weight_value,omitzero"`
	// ID of the unit the case's freight weight is expressed in.
	//
	// Changing the unit relabels the stored weight; the number itself is never converted, so send `freight_weight_value` alongside it when the weight should change too.
	FreightWeightUnitID field.Optional[string] `json:"freight_weight_unit_id,omitzero" validate:"omitempty"`
}

var sampleTrackingNumber = "1Z999AA10123456784"
var sampleUpdateShippingCaseRequest = &UpdateShippingCaseRequest{
	TrackingNumber: field.Some(sampleTrackingNumber),
}

func (*UpdateShippingCaseRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShippingCaseRequest)
}

// Partially updates a shipping case's tracking number and freight quantities.
//
// Fields left out of the request keep their current values. The freight cost and weight recorded here are the case's own actual charge and weight; they do not change the freight billed on the sales order.
type UpdateShippingCaseEndpoint struct{}

func (e *UpdateShippingCaseEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShippingCaseRequest, *apiresource.ShippingCase] {
	return (&apiendpoint.APIEndpoint[*UpdateShippingCaseRequest, *apiresource.ShippingCase]{
		Title:             "Update Shipping Case",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-cases/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		// UpdateShippingCase enforces shipments:update in the service (shipping cases
		// are a facet of shipments). Declared here to match that enforcement.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeShippingCase,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
			return svc.(ShippingCaseSvc).UpdateShippingCase
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingCase,
			Fields:     []string{"carrier", "shipment", "freight_amount.unit", "freight_weight.unit"},
		}),
	})
}
