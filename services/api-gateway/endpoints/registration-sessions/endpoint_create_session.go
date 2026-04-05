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

// The request to create a registration session
type CreateRegistrationSessionRequest struct {
	// The email address for the registration session.
	Email string `json:"email" validate:"required,custom_email"`
	// The plan code for the registration session.
	PlanCode constants.PublicPlanCode `json:"plan_code" validate:"required"`
}

var sampleCreateRegistrationSessionRequest = &CreateRegistrationSessionRequest{
	Email:    apiresource.SampleUserEmail,
	PlanCode: constants.PublicPlanCodeStarter,
}

func (*CreateRegistrationSessionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateRegistrationSessionRequest)
}

type CreateSessionEndpoint struct{}

func (e *CreateSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateRegistrationSessionRequest, *apiresource.CreateSessionResponse] {
	return &apiendpoint.APIEndpoint[*CreateRegistrationSessionRequest, *apiresource.CreateSessionResponse]{
		Title:             "Create Registration Session",
		Description:       "Creates a new registration session for the given email and plan code. Returns the existing session ID if an active session already exists for that email.",
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions",
		ContentType:       "application/json",
		Request:           &CreateRegistrationSessionRequest{},
		Response:          &apiresource.CreateSessionResponse{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateRegistrationSessionRequest) (*apiresource.CreateSessionResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).CreateSession
		},
	}
}
