package regsessionep

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

// Request to update a registration session.
type UpdateSessionRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
	// Step to advance the session to.
	//
	// Must be later than the session's current step; moving backwards is rejected. See the session resource's `step` field for the step order.
	Step field.Optional[constants.RegistrationStep] `json:"step,omitzero"`
	// Session data to merge into the existing session.
	SessionData field.Optional[UpdateSessionDataRequest] `json:"session_data,omitzero"`
}

var sampleUpdateSessionRequest = &UpdateSessionRequest{
	Step: field.Some(constants.RegistrationStepUserDetails),
	SessionData: field.Some(UpdateSessionDataRequest{
		UserName:    field.Some("Jane Smith"),
		AccountName: field.Some("Acme Corp"),
	}),
}

func (*UpdateSessionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSessionRequest)
}

// Mutable form data for a session update.
type UpdateSessionDataRequest struct {
	// Display name for the user.
	UserName field.Optional[string] `json:"user_name,omitzero" validate:"omitempty,max=255"`
	// Display name for the account.
	//
	// Becomes the name of the account created when the registration completes, and must be set before Complete Registration will succeed.
	AccountName field.Optional[string] `json:"account_name,omitzero" validate:"omitempty,max=255"`
	// Billing address line 1.
	BillingAddressLine1 field.Optional[string] `json:"billing_address_line1,omitzero" validate:"omitempty,max=255"`
	// Billing address line 2.
	BillingAddressLine2 field.Optional[string] `json:"billing_address_line2,omitzero" validate:"omitempty,max=255"`
	// Billing address city.
	BillingAddressCity field.Optional[string] `json:"billing_address_city,omitzero" validate:"omitempty,max=255"`
	// Billing address state.
	BillingAddressState field.Optional[string] `json:"billing_address_state,omitzero" validate:"omitempty,max=255"`
	// Billing address postal code.
	BillingAddressPostalCode field.Optional[string] `json:"billing_address_postal_code,omitzero" validate:"omitempty,max=255"`
	// Billing address country as a two-letter country code.
	BillingAddressCountry field.Optional[string] `json:"billing_address_country,omitzero" validate:"omitempty,max=2"`
}

// Partially updates a registration session's step and form data.
//
// Omitted fields are left unchanged, and a session that has already completed can no longer be updated.
type UpdateSessionEndpoint struct{}

func (e *UpdateSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSessionRequest, *apiresource.RegistrationSession] {
	return (&apiendpoint.APIEndpoint[*UpdateSessionRequest, *apiresource.RegistrationSession]{
		Title:             "Update Registration Session",
		Method:            http.MethodPatch,
		Route:             "/v1/auth/registration-sessions/{session_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSessionRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).UpdateSession
		},
	})
}
