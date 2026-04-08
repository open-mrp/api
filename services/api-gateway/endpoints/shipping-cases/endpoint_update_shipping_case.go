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

// UpdateShippingCaseRequest is the request to update a shipping case.
type UpdateShippingCaseRequest struct {
	// The ID of the shipping case to update.
	ShippingCaseID string `path:"id" validate:"required"`
	// The new tracking number.
	TrackingNumber *string `json:"tracking_number" validate:"omitempty,max=255"`
	// The new freight amount value.
	FreightAmountValue *string `json:"freight_amount_value"`
	// The new freight amount unit ID.
	FreightAmountUnitID *string `json:"freight_amount_unit_id" validate:"omitempty,max=191"`
	// The new freight weight value.
	FreightWeightValue *string `json:"freight_weight_value"`
	// The new freight weight unit ID.
	FreightWeightUnitID *string `json:"freight_weight_unit_id" validate:"omitempty,max=191"`
}

var sampleTrackingNumber = "1Z999AA10123456784"
var sampleUpdateShippingCaseRequest = &UpdateShippingCaseRequest{
	TrackingNumber: &sampleTrackingNumber,
}

func (*UpdateShippingCaseRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShippingCaseRequest)
}

type UpdateShippingCaseEndpoint struct{}

func (e *UpdateShippingCaseEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShippingCaseRequest, *apiresource.ShippingCase] {
	return &apiendpoint.APIEndpoint[*UpdateShippingCaseRequest, *apiresource.ShippingCase]{
		Title:             "Update Shipping Case",
		Description:       "Partially updates a shipping case's tracking number and freight quantities.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/shipping-cases/{id}",
		Request:           &UpdateShippingCaseRequest{},
		Response:          &apiresource.ShippingCase{},
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
	}
}
