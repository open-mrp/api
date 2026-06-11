package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a registration flow.
type UpdateRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// IDs of the customer groups to set as this flow's options.
	//
	// Ignored unless `has_customer_group_ids` is `true`.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// IDs of the payment terms to set as this flow's options.
	//
	// Ignored unless `has_payment_term_ids` is `true`.
	PaymentTermIDs []string `json:"payment_term_ids,omitzero"`
	// IDs of the shipping terms to set as this flow's options.
	//
	// Ignored unless `has_shipping_term_ids` is `true`.
	ShippingTermIDs []string `json:"shipping_term_ids,omitzero"`
	// Whether to replace the flow's customer group options with `customer_group_ids`.
	//
	// When `true`, existing options are cleared and replaced (an empty list removes all options). When `false` or omitted, customer group options are left unchanged.
	HasCustomerGroupIDs bool `json:"has_customer_group_ids,omitzero"`
	// Whether to replace the flow's payment term options with `payment_term_ids`.
	//
	// When `true`, existing options are cleared and replaced (an empty list removes all options). When `false` or omitted, payment term options are left unchanged.
	HasPaymentTermIDs bool `json:"has_payment_term_ids,omitzero"`
	// Whether to replace the flow's shipping term options with `shipping_term_ids`.
	//
	// When `true`, existing options are cleared and replaced (an empty list removes all options). When `false` or omitted, shipping term options are left unchanged.
	HasShippingTermIDs bool `json:"has_shipping_term_ids,omitzero"`
}

var sampleUpdateRegistrationFlowRequest = &UpdateRegistrationFlowRequest{
	Name: field.Some("Wholesale Registration Updated"),
}

func (*UpdateRegistrationFlowRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateRegistrationFlowRequest)
}

// Partially updates a registration flow.
type UpdateRegistrationFlowEndpoint struct{}

func (e *UpdateRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateRegistrationFlowRequest, *apiresource.RegistrationFlow] {
	return (&apiendpoint.APIEndpoint[*UpdateRegistrationFlowRequest, *apiresource.RegistrationFlow]{
		Title:             "Update Registration Flow",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/registration-flows/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeRegistrationFlow,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).UpdateRegistrationFlow
		},
	})
}
