package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update a registration flow.
type UpdateRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
	// Display name of the registration flow.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// IDs of the customer groups to offer as this flow's options.
	//
	// Replaces the flow's customer group options; an empty list removes them all. Omit to leave them unchanged.
	CustomerGroupIDs field.Optional[[]string] `json:"customer_group_ids,omitzero"`
	// IDs of the payment terms to offer as this flow's options.
	//
	// Replaces the flow's payment term options; an empty list removes them all. Omit to leave them unchanged.
	PaymentTermIDs field.Optional[[]string] `json:"payment_term_ids,omitzero"`
	// IDs of the shipping terms to offer as this flow's options.
	//
	// Replaces the flow's shipping term options; an empty list removes them all. Omit to leave them unchanged.
	ShippingTermIDs field.Optional[[]string] `json:"shipping_term_ids,omitzero"`
}

var sampleUpdateRegistrationFlowRequest = &UpdateRegistrationFlowRequest{
	Name:           field.Some("Wholesale Registration Updated"),
	PaymentTermIDs: field.Some([]string{apiresource.SamplePaymentTermID}),
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeRegistrationFlow,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).UpdateRegistrationFlow
		},
	})
}
