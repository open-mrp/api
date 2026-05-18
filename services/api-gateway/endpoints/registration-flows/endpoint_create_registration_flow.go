package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a registration flow.
type CreateRegistrationFlowRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Customer group IDs.
	CustomerGroupIDs []string `json:"customer_group_ids"`
	// Payment term IDs.
	PaymentTermIDs []string `json:"payment_term_ids"`
	// Shipping term IDs.
	ShippingTermIDs []string `json:"shipping_term_ids"`
}

var sampleCreateRegistrationFlowRequest = &CreateRegistrationFlowRequest{
	Name:             "Wholesale Registration",
	CustomerGroupIDs: []string{"cgrp_01abc"},
	PaymentTermIDs:   []string{"pt_01abc"},
	ShippingTermIDs:  []string{"st_01abc"},
}

func (*CreateRegistrationFlowRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateRegistrationFlowRequest)
}

// Creates a registration flow.
type CreateRegistrationFlowEndpoint struct{}

func (e *CreateRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateRegistrationFlowRequest, *apiresource.RegistrationFlow] {
	return (&apiendpoint.APIEndpoint[*CreateRegistrationFlowRequest, *apiresource.RegistrationFlow]{
		Title:             "Create Registration Flow",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/registration-flows",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).CreateRegistrationFlow
		},
		LocationFunc: func(resp *apiresource.RegistrationFlow) string {
			return "/v1/sales/registration-flows/" + resp.ID
		},
	})
}
