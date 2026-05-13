package httpgroup

import (
	"fmt"

	regsessionep "github.com/augno/api/services/api-gateway/endpoints/registration-sessions"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/middleware"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type RegistrationSessionsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type RegistrationSessionsEndpointGroupConfig struct {
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

	createSessionEndpoint := (&regsessionep.CreateSessionEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	resendEmailEndpoint := (&regsessionep.ResendEmailEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	verifyTokenEndpoint := (&regsessionep.VerifyTokenEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	getSessionEndpoint := (&regsessionep.RetrieveSessionEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	createUserEndpoint := (&regsessionep.CreateUserEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	updateSessionEndpoint := (&regsessionep.UpdateSessionEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	listSessionsEndpoint := (&regsessionep.ListSessionsEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	setupBillingEndpoint := (&regsessionep.SetupBillingEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	confirmPaymentEndpoint := (&regsessionep.ConfirmPaymentEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)
	completeRegistrationEndpoint := (&regsessionep.CompleteRegistrationEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, svc)

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
