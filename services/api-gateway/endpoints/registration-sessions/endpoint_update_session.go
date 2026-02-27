package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// The request to update a registration session
type UpdateSessionRequest struct {
	// The session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
	// The step to advance the session to.
	Step *constants.RegistrationStep `json:"step,omitempty"`
	// The session data to merge into the existing session data.
	SessionData *UpdateSessionDataRequest `json:"session_data,omitempty"`
}

var sampleUpdateSessionRequest = &UpdateSessionRequest{
	Step: new(constants.RegistrationStepUserDetails),
	SessionData: &UpdateSessionDataRequest{
		UserName:    new("Jane Smith"),
		AccountName: new("Acme Corp"),
	},
}

func (*UpdateSessionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSessionRequest)
}

// UpdateSessionDataRequest holds the mutable form data fields for updating a session
type UpdateSessionDataRequest struct {
	// Display name for the user.
	UserName *string `json:"user_name,omitempty"`
	// Display name for the account.
	AccountName *string `json:"account_name,omitempty"`
	// Billing address line 1.
	BillingAddressLine1 *string `json:"billing_address_line1,omitempty"`
	// Billing address line 2.
	BillingAddressLine2 *string `json:"billing_address_line2,omitempty"`
	// Billing address city.
	BillingAddressCity *string `json:"billing_address_city,omitempty"`
	// Billing address state.
	BillingAddressState *string `json:"billing_address_state,omitempty"`
	// Billing address postal code.
	BillingAddressPostalCode *string `json:"billing_address_postal_code,omitempty"`
	// Billing address country.
	BillingAddressCountry *string `json:"billing_address_country,omitempty"`
}

const updateSessionDescription string = `Updates an in-progress registration session's step and/or form data.
Only provided fields are applied; omitted fields are left unchanged.`

type UpdateSessionEndpoint struct{}

func (e *UpdateSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSessionRequest, *apiresource.RegistrationSession] {
	return &apiendpoint.APIEndpoint[*UpdateSessionRequest, *apiresource.RegistrationSession]{
		Title:             "Update Registration Session",
		Description:       updateSessionDescription,
		Method:            http.MethodPatch,
		Route:             "/v1/auth/registration-sessions/{session_id}",
		ContentType:       "application/json",
		Request:           &UpdateSessionRequest{},
		Response:          &apiresource.RegistrationSession{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSessionRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).UpdateSession
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
