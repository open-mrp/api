package portalregsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to start or resume a customer-portal registration session.
type CreateOrResumePortalRegistrationSessionRequest struct {
	// The seller's portal slug to register into.
	SellerSlug string `json:"seller_slug" validate:"required"`
}

var sampleCreateOrResumePortalRegistrationSessionRequest = &CreateOrResumePortalRegistrationSessionRequest{
	SellerSlug: "acme-inc",
}

func (*CreateOrResumePortalRegistrationSessionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateOrResumePortalRegistrationSessionRequest)
}

// Starts a new customer-portal registration session for the authenticated buyer, or resumes the buyer's existing in-progress session for the same seller.
//
// Registering into a seller's portal is a multi-step flow; the session tracks progress so a half-finished registration can be resumed instead of leaving the buyer stuck.
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
