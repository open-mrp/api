package portalregsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// PortalRegistrationSessionDataInput is the scratch form data saved on a registration session as the buyer advances.
type PortalRegistrationSessionDataInput struct {
	CustomerName      string `json:"customer_name,omitzero"`
	CustomerNumber    string `json:"customer_number,omitzero"`
	CustomerGroupID   string `json:"customer_group_id,omitzero"`
	PaymentTermID     string `json:"payment_term_id,omitzero"`
	ShippingTermID    string `json:"shipping_term_id,omitzero"`
	Phone             string `json:"phone,omitzero"`
	AddressName       string `json:"address_name,omitzero"`
	AddressStreet1    string `json:"address_street_1,omitzero"`
	AddressStreet2    string `json:"address_street_2,omitzero"`
	AddressLocality   string `json:"address_locality,omitzero"`
	AddressState      string `json:"address_state,omitzero"`
	AddressPostalCode string `json:"address_postal_code,omitzero"`
	AddressCountry    string `json:"address_country,omitzero"`
}

// Request to advance a portal registration session to the next step.
type UpdatePortalRegistrationSessionRequest struct {
	// Portal registration session ID.
	ID string `path:"id" validate:"required"`
	// The step to advance to. Steps are forward-only.
	Step constants.PortalRegistrationStep `json:"step" validate:"required"`
	// The accumulated form data to save on the session.
	SessionData *PortalRegistrationSessionDataInput `json:"session_data,omitzero"`
	// Whether the buyer is linking an existing customer vs. creating a new one.
	IsExistingCustomer field.Optional[bool] `json:"is_existing_customer,omitzero"`
}

// Advances the buyer's registration session to the given step and saves the accumulated form data. Steps are forward-only; a completed or abandoned session cannot be updated.
type UpdatePortalRegistrationSessionEndpoint struct{}

func (e *UpdatePortalRegistrationSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession] {
	return (&apiendpoint.APIEndpoint[*UpdatePortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession]{
		Title:             "Update Portal Registration Session",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/portal-registration-sessions/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePortalRegistrationSession,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
			return svc.(PortalRegistrationSessionSvc).Update
		},
	})
}
