package shippingcaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a shipping case.
type UpdateShippingCaseRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
	// Tracking number.
	TrackingNumber *string `json:"tracking_number" nullable:"false" validate:"omitempty,max=255"`
	// Freight amount value.
	FreightAmountValue *string `json:"freight_amount_value" nullable:"false"`
	// Freight amount unit ID.
	FreightAmountUnitID *string `json:"freight_amount_unit_id" nullable:"false" validate:"omitempty"`
	// Freight weight value.
	FreightWeightValue *string `json:"freight_weight_value" nullable:"false"`
	// Freight weight unit ID.
	FreightWeightUnitID *string `json:"freight_weight_unit_id" nullable:"false" validate:"omitempty"`
}

var sampleTrackingNumber = "1Z999AA10123456784"
var sampleUpdateShippingCaseRequest = &UpdateShippingCaseRequest{
	TrackingNumber: &sampleTrackingNumber,
}

func (*UpdateShippingCaseRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShippingCaseRequest)
}

// Partially updates a shipping case's tracking number and freight quantities.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
			return svc.(ShippingCaseSvc).UpdateShippingCase
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingCase,
			Fields:     []string{"carrier", "shipment", "freight_amount.unit", "freight_weight.unit"},
		}),
	})
}
