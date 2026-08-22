package httpgroup

import (
	"fmt"

	regsessionep "github.com/open-mrp/api/services/api-gateway/endpoints/registration-sessions"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	"github.com/open-mrp/api/services/api-gateway/internal/middleware"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type RegistrationSessionsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type RegistrationSessionsEndpointGroupConfig struct {
	// AuthClient (required) is the auth-service gRPC client.
	AuthClient *grpcclient.AuthServiceClient
}

func (c *RegistrationSessionsEndpointGroupConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("registration sessions endpoint group: auth client is required")
	}
	return nil
}

func (*RegistrationSessionsEndpointGroup) Materialize(config *RegistrationSessionsEndpointGroupConfig) *RegistrationSessionsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := regsessionep.NewRegistrationSessionSvc(&regsessionep.RegistrationSessionSvcConfig{
		AuthClient: config.AuthClient.Client,
	})

	authMw := middleware.AuthMiddleware(&middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Registration Sessions",
		Description:  "Create and manage registration sessions for the multi-step registration flow.",
		ResourceType: &apiresource.CreateSessionResponse{},
	}

	createSessionEndpoint := apiendpoint.From(&regsessionep.CreateSessionEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	resendEmailEndpoint := apiendpoint.From(&regsessionep.ResendEmailEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	verifyTokenEndpoint := apiendpoint.From(&regsessionep.VerifyTokenEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	getSessionEndpoint := apiendpoint.From(&regsessionep.RetrieveSessionEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	createUserEndpoint := apiendpoint.From(&regsessionep.CreateUserEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	updateSessionEndpoint := apiendpoint.From(&regsessionep.UpdateSessionEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	listSessionsEndpoint := apiendpoint.From(&regsessionep.ListSessionsEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	setupBillingEndpoint := apiendpoint.From(&regsessionep.SetupBillingEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	confirmPaymentEndpoint := apiendpoint.From(&regsessionep.ConfirmPaymentEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)
	completeRegistrationEndpoint := apiendpoint.From(&regsessionep.CompleteRegistrationEndpoint{}).WithMiddleware(authMw).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		createSessionEndpoint,
		resendEmailEndpoint,
		verifyTokenEndpoint,
		getSessionEndpoint,
		createUserEndpoint,
		updateSessionEndpoint,
		listSessionsEndpoint,
		setupBillingEndpoint,
		confirmPaymentEndpoint,
		completeRegistrationEndpoint,
	}

	return &RegistrationSessionsEndpointGroup{inner}
}
