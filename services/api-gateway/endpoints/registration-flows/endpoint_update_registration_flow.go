package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateRegistrationFlowRequest is the request to partially update a registration flow.
type UpdateRegistrationFlowRequest struct {
	// The ID of the registration flow to update.
	RegistrationFlowID string `path:"id" validate:"required"`
	// The display name of the registration flow.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The IDs of the customer groups associated with this registration flow.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// The IDs of the payment terms associated with this registration flow.
	PaymentTermIDs []string `json:"payment_term_ids,omitempty"`
	// The IDs of the shipping terms associated with this registration flow.
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

type UpdateRegistrationFlowEndpoint struct{}

func (e *UpdateRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateRegistrationFlowRequest, *apiresource.RegistrationFlow] {
	return &apiendpoint.APIEndpoint[*UpdateRegistrationFlowRequest, *apiresource.RegistrationFlow]{
		Title:             "Update Registration Flow",
		Description:       "Partially updates a registration flow.",
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
	}
}
