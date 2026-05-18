package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a registration flow.
type UpdateRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Customer group IDs.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// Payment term IDs.
	PaymentTermIDs []string `json:"payment_term_ids,omitempty"`
	// Shipping term IDs.
	ShippingTermIDs []string `json:"shipping_term_ids,omitempty"`
	// Whether to replace customer groups.
	HasCustomerGroupIDs bool `json:"has_customer_group_ids,omitempty"`
	// Whether to replace payment terms.
	HasPaymentTermIDs bool `json:"has_payment_term_ids,omitempty"`
	// Whether to replace shipping terms.
	HasShippingTermIDs bool `json:"has_shipping_term_ids,omitempty"`
}

func updateStrPtr(s string) *string { return &s }

var sampleUpdateRegistrationFlowRequest = &UpdateRegistrationFlowRequest{
	Name: updateStrPtr("Wholesale Registration Updated"),
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
		Request:           &UpdateRegistrationFlowRequest{},
		Response:          &apiresource.RegistrationFlow{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).UpdateRegistrationFlow
		},
	}).WithDocSource(e)
}
