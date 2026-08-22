package portalregsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to start or resume a customer-portal registration session.
type CreateOrResumePortalRegistrationSessionRequest struct {
	// The portal slug of the seller the buyer is registering with.
	SellerSlug string `json:"seller_slug" validate:"required"`
}

var sampleCreateOrResumePortalRegistrationSessionRequest = &CreateOrResumePortalRegistrationSessionRequest{
	SellerSlug: "acme-inc",
}

func (*CreateOrResumePortalRegistrationSessionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateOrResumePortalRegistrationSessionRequest)
}

// Starts a customer-portal registration for the authenticated buyer, or resumes the one they already have with this seller.
//
// Registering into a seller's portal is a multi-step flow, and the session carries the progress so a half-finished registration is never lost. If the buyer has an unfinished session with this seller that is still inside its seven-day resume window, that session comes back with its saved step and form data; otherwise a new one starts at the `customer_details` step. Completed, abandoned, and expired sessions are never resumed.
type CreateOrResumePortalRegistrationSessionEndpoint struct{}

func (e *CreateOrResumePortalRegistrationSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateOrResumePortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession] {
	return (&apiendpoint.APIEndpoint[*CreateOrResumePortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession]{
		Title:             "Start or Resume Portal Registration",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/portal-registration-sessions",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePortalRegistrationSession,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateOrResumePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
			return svc.(PortalRegistrationSessionSvc).CreateOrResume
		},
	})
}
