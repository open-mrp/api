package portalregsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// The form data to save on a registration session as the buyer advances.
//
// These values are what the registration is completed from, so send everything collected so far on each update — the stored data is replaced outright rather than merged.
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

// Request to save a buyer's progress on a portal registration session.
type UpdatePortalRegistrationSessionRequest struct {
	// Portal registration session ID.
	ID string `path:"id" validate:"required"`
	// The step the buyer has reached.
	//
	// Steps only move forward: sending an earlier step than the session has already reached is rejected, while re-sending the current step saves data without advancing.
	Step constants.PortalRegistrationStep `json:"step" validate:"required"`
	// The form data collected so far.
	//
	// This replaces the session's saved data rather than merging into it, so anything left out is cleared.
	SessionData *PortalRegistrationSessionDataInput `json:"session_data,omitzero"`
	// Whether the buyer is joining a customer the seller already has, rather than creating a new one.
	//
	// This decides what completing the registration does: joining an existing customer links the buyer to the customer matching `customer_number`, while a new customer is built from the rest of the session data. Like the session data it is stored as sent, so re-send it on every update to keep the choice.
	IsExistingCustomer field.Optional[bool] `json:"is_existing_customer,omitzero"`
}

// Advances the buyer's registration session and saves the data entered so far.
//
// Each update writes the session's step, form data, and existing-customer choice as sent, so send the full picture every time rather than just the newly-entered fields. Steps only move forward, and a session that has already been completed or abandoned can no longer be updated.
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
