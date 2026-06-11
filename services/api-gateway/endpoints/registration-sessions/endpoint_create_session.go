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

// Request to create a registration session.
type CreateRegistrationSessionRequest struct {
	// Email address of the registering user.
	//
	// A verification email is sent to this address to start the registration.
	Email string `json:"email" validate:"required,custom_email,max=255"`
	// Code of the pricing plan to register for.
	//
	// Free plans skip the payment step; paid plans require a payment method to be collected and confirmed before the registration can complete.
	PlanCode constants.PublicPlanCode `json:"plan_code" validate:"required"`
}

var sampleCreateRegistrationSessionRequest = &CreateRegistrationSessionRequest{
	Email:    apiresource.SampleUserEmail,
	PlanCode: constants.PublicPlanCodeStarter,
}

func (*CreateRegistrationSessionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateRegistrationSessionRequest)
}

// Starts a self-serve registration session and sends a verification email.
//
// If an active session already exists for the email, the existing session's ID is returned instead of creating a new one.
type CreateSessionEndpoint struct{}

func (e *CreateSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateRegistrationSessionRequest, *apiresource.CreateSessionResponse] {
	return (&apiendpoint.APIEndpoint[*CreateRegistrationSessionRequest, *apiresource.CreateSessionResponse]{
		Title:             "Create Registration Session",
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateRegistrationSessionRequest) (*apiresource.CreateSessionResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).CreateSession
		},
	})
}
